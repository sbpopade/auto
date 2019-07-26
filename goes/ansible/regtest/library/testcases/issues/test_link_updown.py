#!/usr/bin/python
""" Verify vnetd daemon """

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
import time

from collections import OrderedDict

from ansible.module_utils.basic import AnsibleModule

DOCUMENTATION = """
---
module: test_link_updown
author: Platina Systems
short_description: Module to test vnetd daemon is up.
description:
    Module to test vnetd daemon is up.
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
    frequency:
      description:
        - The no of times the interface is to be moved into a namespace.
      required: False
      type: str
    spine_list:
      description:
        - List of all spine switches.
      required: False
      type: list
      default: []
    leaf_list:
      description:
        - List of all leaf switches.
      required: False
      type: list
      default: []
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
- name: Execute and verify port links
  test_link_updown:
    switch_name: "{{ inventory_hostname }}"
    hash_name: "{{ hostvars['server_emulator']['hash_name'] }}"
    log_dir_path: "{{ issues_log_dir }}"
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

    if out:
        HASH_DICT[key] = out[:512] if len(out.encode('utf-8')) > 512 else out
    else:
        HASH_DICT[key] = out

    return out


def verify_goes_status(module):
    """
    Method to restart goes frequent times on invader.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS
    failure_summary = ''

    switch_name = module.params['switch_name']

    # Get the GOES status info
    status = execute_commands(module, 'goes status').splitlines()
    for line in status:
        if 'Not OK' in line:
            RESULT_STATUS = False
            failure_summary += 'On switch {} '.format(switch_name)
            failure_summary += 'GOES status is Not OK.\n'
            failure_summary += '{}\n'.format(line)

    return failure_summary


def move_interface(module):
    """
    Method to verify dumps in log files on invader.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS
    failure_summary = ''

    switch_name = module.params['switch_name']
    eth = module.params['eth']
    spine_list = module.params['spine_list']
    leaf_list = module.params['leaf_list']
    frequency = module.params['frequency']

    is_spine = True if switch_name in spine_list else False

    intf1 = 2
    intf2 = 3

    if is_spine:
        target = leaf_list[0][-2::]
	clr = 'red'
	anum = 16
	a, b = intf2, intf1
    else:
        target = spine_list[0][-2::]
	clr = 'blue'
	anum = 15
	a, b = intf1, intf2

    cmd = 'ip netns add {}'.format(clr)
    execute_commands(module, cmd)
    cmd = 'ip link set xeth{} netns {}'.format(eth, clr)
    execute_commands(module, cmd)
    cmd = 'ip netns set {} {}'.format(clr, anum)
    execute_commands(module, cmd)

    status = True

    if switch_name:
        cmd = 'ip netns exec {} ip link set up xeth{}'.format(clr, eth)
        execute_commands(module, cmd)
        cmd = 'ip netns exec {} ip add add 10.1.0.{} peer 10.1.0.{}/31 dev xeth{}'.format(clr, a, b, eth)
        execute_commands(module, cmd)
	if is_spine:
		cmd = 'ip netns exec {} ping -c 100 10.1.0.{}'.format(clr, b)
		ping = execute_commands(module, cmd)
		if '0% packet loss' in ping:
		    RESULT_STATUS = False
		    failure_summary += 'On switch {} '.format(switch_name)
		    failure_summary += 'ping replies are coming even after setting down '
		    failure_summary += 'the interface on destination switch.\n'
		elif '100% packet loss' in ping:
		    failure_summary += 'From switch {} '.format(switch_name)
		    failure_summary += 'the interface on destination switch is not pingable.\n'
	else:
		cmd = " ip netns exec {} ip link set down xeth{}".format(clr, eth)
		execute_commands(module, cmd)
		cmd = " ip netns exec {} ip link set up xeth{}".format(clr, eth)
                execute_commands(module, cmd)



    time.sleep(10)
    failure_summary += verify_goes_status(module)
    cmd = 'ip netns del {}'.format(switch_name)
    execute_commands(module, cmd)
    execute_commands(module, "ifdown -a --allow vnet")
    execute_commands(module, "ifup -a --allow vnet")
#    execute_commands(module, "exit")
    cmd = 'goes restart'
    execute_commands(module, cmd)
    time.sleep(10)
    failure_summary += verify_goes_status(module)

    HASH_DICT['result.detail'] = failure_summary


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            eth=dict(required=False, type='str'),
            frequency=dict(required=False, type='int', default=1),
            spine_list=dict(required=False, type='list', default=[]),
            leaf_list=dict(required=False, type='list', default=[]),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global RESULT_STATUS, HASH_DICT

    move_interface(module)

    # Calculate the entire test result
    HASH_DICT['result.status'] = 'Passed' if RESULT_STATUS else 'Failed'

    # Create a log file
    log_file_path = module.params['log_dir_path']
    log_file_path += '/{}.log'.format(module.params['hash_name'])
    log_file = open(log_file_path, 'a')
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


