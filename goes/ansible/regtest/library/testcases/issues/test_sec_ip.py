#!/usr/bin/python
""" Test Secondary IP """

#
# This file is part of Ansible
#
# Ansible is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Ansible is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Ansible. If not, see <http://www.gnu.org/licenses/>.
#

import shlex

from collections import OrderedDict

from ansible.module_utils.basic import AnsibleModule

DOCUMENTATION = """
---
module: test_sec_ip
author: Platina Systems
short_description: Module to assign secondary IP to a interface.
description:
    Module to assign secondary IP and check FIB entries.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    eth:
      description:
        - Name of the interface which is to be moved inside the container.
      required: False
      type: str
    hash_name:
      description:
        - Name of the hash in which to store the result in redis.
      required: False
      type: str
    log_dir_path:
      description:
        - Path to log directory where logs will be stored.
      required: False
      type: str
"""

EXAMPLES = """
- name: Move interface and verify FIB entries
  test_sec_ip:
    switch_name: "{{ inventory_hostname }}"
    hash_name: "{{ hostvars['server_emulator']['hash_name'] }}"
    log_dir_path: "{{ issue_dir_path }}"
"""

RETURN = """
hash_dict:
  description: Dictionary containing key value pairs to store in hash.
  returned: always
  type: dict
"""

RESULT_STATUS = True
HASH_DICT = OrderedDict()


def run_cli(module, cli):
    """
    Method to execute the cli command on the target node(s) and
    returns the output.
    :param module: The Ansible module to fetch input parameters.
    :param cli: The complete cli string to be executed on the target node(s).
    :return: Output/Error or None depending upon the response from cli.
    """
    cli = shlex.split(cli)
    rc, out, err = module.run_command(cli)

    if out:
        return out.rstrip()
    elif err:
        return err.rstrip()
    else:
        return None


def execute_commands(module, cmd):
    """
    Method to execute given commands and return the output.
    :param module: The Ansible module to fetch input parameters.
    :param cmd: Command to execute.
    :return: Output of the commands.
    """
    global HASH_DICT

    out = run_cli(module, cmd)

    # Store command prefixed with exec time as key and
    # command output as value in the hash dictionary
    exec_time = run_cli(module, 'date +%Y%m%d%T')
    key = '{0} {1} {2}'.format(module.params['switch_name'], exec_time, cmd)

    HASH_DICT[key] = out
    return out

import time
def verify_fib(module):
    """
    Method to bring up and bring down docker containers.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    eth = module.params['eth']

    cmd = 'ifconfig xeth{}'.format(eth)
    out = execute_commands(module, cmd).splitlines()

    # Assign secondary IP to interface
    ip2 = '192.168.1.{}'.format(eth)
    cmd = 'ifconfig xeth{}:0 {} netmask 255.255.255.0 up'.format(eth, ip2)
    execute_commands(module, cmd)
    time.sleep(10)
    cmd = 'ip add sh xeth{}'.format(eth)
    out = execute_commands(module, cmd)
    if ip2 not in out:
        RESULT_STATUS = False
        failure_summary += 'On switch {} '.format(switch_name)
        failure_summary += 'both the IPs are not showing up\n'

    cmd = 'goes vnet show ip fib'
    out = execute_commands(module, cmd)
    if ip2 not in out:
        RESULT_STATUS = False
        failure_summary += 'On switch {} '.format(switch_name)
        failure_summary += 'FIB entry for both the IPs are not showing up '
        failure_summary += 'before moving interface to name space.\n'

    # Moving interface to name space
    cmd = 'ip netns add red'
    execute_commands(module, cmd)
    cmd = 'ip link set xeth{} netns red'.format(eth)
    execute_commands(module, cmd)
    cmd = 'ip netns set red 20'
    execute_commands(module, cmd)
    time.sleep(10)
    cmd = 'ip netns exec red ip add sh'
    execute_commands(module, cmd)
    cmd = 'ip netns exec red ip link set up xeth{}'.format(eth)
    execute_commands(module, cmd)
    cmd = 'goes vnet show ip fib'
    out = execute_commands(module, cmd)
    if 'xeth{}'.format(eth) in out:
        RESULT_STATUS = False
        failure_summary += 'On switch {} '.format(switch_name)
        failure_summary += 'FIB entry for both the IPs are showing up '
        failure_summary += 'even after moving interface to name space.\n'

    execute_commands(module, cmd)
    time.sleep(5)
    cmd = 'ip netns del red'
    execute_commands(module, cmd)

    HASH_DICT['result.detail'] = failure_summary

    # Get the GOES status info
    execute_commands(module, "ifdown -a --allow vnet")
    execute_commands(module, "ifup -a --allow vnet")
    execute_commands(module, 'goes restart')
    time.sleep(10)
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            eth=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS

    verify_fib(module)

    # Calculate the entire test result
    HASH_DICT['result.status'] = 'Passed' if RESULT_STATUS else 'Failed'

    # Create a log file
    log_file_path = module.params['log_dir_path']
    log_file_path += '/{}_'.format(module.params['hash_name']) + '.log'
    log_file = open(log_file_path, 'w')
    for key, value in HASH_DICT.iteritems():
        log_file.write(key)
        log_file.write('\n')
        log_file.write(str(value))
        log_file.write('\n')
        log_file.write('\n')

    log_file.close()

    # Exit the module and return the required JSON.
    module.exit_json(
        hash_dict=HASH_DICT,
        log_file_path=log_file_path
    )

if __name__ == '__main__':
    main()


