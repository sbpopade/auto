#!/usr/bin/python
""" Test/Verify BGP MED (Multi-Exit Discriminator) """

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
module: test_bgp_med
author: Platina Systems
short_description: Module to test and verify bgp configurations.
description:
    Module to test and verify bgp configurations and log the same.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    config_file:
      description:
        - BGP config which have been added.
      required: False
      type: str
    med_switch:
      description:
        - Name of the switch on which med config has been added.
      required: False
      type: str
    package_name:
      description:
        - Name of the package installed (e.g. quagga/frr/bird).
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
- name: Verify bgp peering med
  test_bgp_med:
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

    if ('service' in cmd and 'restart' in cmd) or module.params['dry_run_mode']:
        out = None
    else:
        out = run_cli(module, cmd)

    # Store command prefixed with exec time as key and
    # command output as value in the hash dictionary
    exec_time = run_cli(module, 'date +%Y%m%d%T')
    key = '{0} {1} {2}'.format(module.params['switch_name'], exec_time, cmd)
    HASH_DICT[key] = out

    return out


def verify_bgp_med(module):
    """
    Method to verify bgp med.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    package_name = module.params['package_name']
    med_switch = module.params['med_switch']
    config_file = module.params['config_file'].splitlines()
    delay = module.params['delay']
    retries = module.params['retries']

    # Get the current/running configurations
    execute_commands(module, "vtysh -c 'sh running-config'")

    # Restart and check package status
    execute_commands(module, 'service {} restart'.format(package_name))
    execute_commands(module, 'service {} status'.format(package_name))

    if switch_name == med_switch:
        # Get all bgp routes
        while(retries):
            failure_summary = ''
            RESULT_STATUS = True
            cmd = "vtysh -c 'sh ip bgp'"
            bgp_out = execute_commands(module, cmd)

            if bgp_out:
                for line in config_file:
                    line = line.strip()
                    if line.startswith('ip prefix-list'):
                        ip = line.split().pop()
                        if ip not in bgp_out:
                            RESULT_STATUS = False
                            failure_summary += 'On switch {} '.format(switch_name)
                            failure_summary += 'bgp route for network {} '.format(ip)
                            failure_summary += 'is not present in the '
                            failure_summary += 'output of command {}\n'.format(cmd)

                    if line.startswith('set metric'):
                        med = line.split().pop()
                        med = ' {} '.format(med)
                        if med not in bgp_out:
                            RESULT_STATUS = False
                            failure_summary += 'On switch {} '.format(switch_name)
                            failure_summary += 'MED value{}is not present '.format(med)
                            failure_summary += 'in output of command {}\n'.format(cmd)
            else:
                RESULT_STATUS = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'bgp med cannot be verified since '
                failure_summary += 'output of command {} '.format(cmd)
                failure_summary += 'is None'

            if not RESULT_STATUS:
                time.sleep(delay)
                retries -= 1
            else:
                break

    HASH_DICT['result.detail'] = failure_summary

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            config_file=dict(required=False, type='str'),
            med_switch=dict(required=False, type='str'),
            package_name=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
            delay=dict(required=False, type='int', default=10),
            retries=dict(required=False, type='int', default=6),
            dry_run_mode=dict(required=False, type='bool', default=False),
        )
    )

    global HASH_DICT, RESULT_STATUS
    if module.params['dry_run_mode']:

        package_name = module.params['package_name']
        cmds_list = []

        execute_commands(module, "vtysh -c 'sh running-config'")

        # Restart and check package status
        execute_commands(module, 'service {} restart'.format(package_name))
        execute_commands(module, 'service {} status'.format(package_name))
        cmd = "vtysh -c 'sh ip bgp'"
        execute_commands(module, cmd)
        execute_commands(module, 'goes status')

        for key, value in HASH_DICT.iteritems():
            cmds_list.append(key)

        # Exit the module and return the required JSON.
        module.exit_json(
            cmds=cmds_list
        )
    else:
        verify_bgp_med(module)

        # Calculate the entire test result
        HASH_DICT['result.status'] = 'Passed' if RESULT_STATUS else 'Failed'

        # Create a log file
        log_file_path = module.params['log_dir_path']
        log_file_path += '/{}.log'.format(module.params['hash_name'])
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

