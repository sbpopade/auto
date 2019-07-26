#!/usr/bin/python
""" Docker Up Down """

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
module: test_updown_fib
author: Platina Systems
short_description: Module to bring up and bring down docker containers.
description:
    Module to bring up and bring down docker containers.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    container_name:
      description:
        - Name of the container.
      required: False
      type: str
    eth:
      description:
        - Name of the interface which is to be moved inside the container.
      required: False
      type: str
    frequency:
      description:
        - The no of times the interface is to be moved in and out of container.
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
  test_updown_fib:
    switch_name: "{{ inventory_hostname }}"
    hash_name: "{{ hostvars['server_emulator']['hash_name'] }}"
    log_dir_path: "{{ log_dir_path }}"
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


def verify_fib(module):
    """
    Method to bring up and bring down docker containers.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    container_name = module.params['container_name']
    frequency = module.params['frequency']
    eth = module.params['eth']

    d_move = '~/./docker_move.sh'

    if frequency != 1:
        ip = '10.15.0.1'
        for i in range(0, frequency):
            # Bring up given interfaces in the docker container
            cmd = '{} up {} xeth{} {}/24'.format(d_move, container_name, eth, ip)
            out = execute_commands(module, cmd)
            if out:
                RESULT_STATUS = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'unable to move the interface xeth{} into docker\n'.format(eth)

            cmd = 'goes vnet show ip fib'
            out = execute_commands(module, cmd)
            if ip not in out:
                RESULT_STATUS = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'FIB entry is not showing up for interface {}\n'.format(eth)

            # Bring down the interface from the docker container
            cmd = '{} down {} xeth{}'.format(d_move, container_name, eth)
            out = execute_commands(module, cmd)
            if out:
                RESULT_STATUS = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'Command- {}\n'.format(cmd)
                failure_summary += 'Down output- {}\n'.format(out)

            cmd = 'goes vnet show ip fib'
            out = execute_commands(module, cmd)
            if ip in out:
                RESULT_STATUS = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'the interface is not moved out from container {}\n'.format(eth)

    else:
        ip = '192.168.120.{}'.format(eth)
        # Bring up given interfaces in the docker container
        cmd = '{} up {} xeth{}'.format(d_move, container_name, eth)
        execute_commands(module, cmd)
	import time
	time.sleep(10)
        cmd = 'goes vnet show ip fib'
        out = execute_commands(module, cmd)
        if ip not in out:
            RESULT_STATUS = False
            failure_summary += 'On switch {} '.format(switch_name)
            failure_summary += 'FIB entry is not showing up for interface {}\n'.format(eth)
        else:
            out = out.splitlines()
            for line in out:
                if ip in line:
                    line = line.strip()
                    t_id = line.split()[0]

                    cmd = 'goes vnet show ip fib sum'
                    out = execute_commands(module, cmd)
                    if t_id not in out:
                        RESULT_STATUS = False
                        failure_summary += 'On switch {} '.format(switch_name)
                        failure_summary += 'FIB summary is not appropriate for interface {}\n'.format(eth)

        # Bring down the interface from the docker container
        cmd = '{} down {} xeth{}'.format(d_move, container_name, eth)
        out = execute_commands(module, cmd)
        if out:
            RESULT_STATUS = False
            failure_summary += 'On switch {} '.format(switch_name)
            failure_summary += 'Command- {}\n'.format(cmd)
            failure_summary += 'Down output- {}\n'.format(out)

    HASH_DICT['result.detail'] = failure_summary

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            container_name=dict(required=False, type='str'),
            eth=dict(required=False, type='str'),
            frequency=dict(required=False, type='int', default=1),
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


