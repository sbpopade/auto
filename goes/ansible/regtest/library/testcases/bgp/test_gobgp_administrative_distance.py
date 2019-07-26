#!/usr/bin/python
""" Test/Verify GOBGP Administrative Distance """

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
module: test_gobgp_administrative_distance
author: Platina Systems
short_description: Module to test and verify gobgp administrative distance.
description:
    Module to test and verify gobgp configurations and log the same.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    package_name:
      description:
        - Name of the package installed (e.g. gobgpd).
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
- name: Verify gobgp administrative distance
  test_gobgp_administrative_distance:
    switch_name: "{{ inventory_hostname }}"
    spine_list: "{{ groups['spine'] }}"
    leaf_list: "{{ groups['leaf'] }}"
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

    if 'service' in cmd and 'restart' in cmd:
        out = None
    else:
        out = run_cli(module, cmd)

    # Store command prefixed with exec time as key and
    # command output as value in the hash dictionary
    exec_time = run_cli(module, 'date +%Y%m%d%T')
    key = '{0} {1} {2}'.format(module.params['switch_name'], exec_time, cmd)
    HASH_DICT[key] = out

    return out


def verify_gobgp_administrative_distance(module):
    """
    Method to verify gobgp administrative distance.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    package_name = module.params['package_name']
    spine_list = module.params['spine_list']
    leaf_list = module.params['leaf_list']
    delay = module.params['delay']
    retries = module.params['retries']
    routes_to_check = []

    # Get the gobgp config
    execute_commands(module, 'cat /etc/gobgp/gobgpd.conf')

    # Restart and check package status
    execute_commands(module, 'service {} restart'.format(package_name))
    execute_commands(module, 'service {} status'.format(package_name))

    # Get gobgp neighbors
    cmd = 'gobgp nei'
    execute_commands(module, cmd)

    while(retries):
	# Get all ip routes
	cmd = "vtysh -c 'sh ip route'"
    	all_routes = execute_commands(module, cmd)

        RESULT_STATUS = True
        failure_summary = ''
        if all_routes:
            switch_list = leaf_list if switch_name in spine_list else spine_list
            for switch in switch_list:
                routes_to_check.append('192.168.{}.1/32'.format(switch[-2::]))

            routes_to_check.append('192.168.{}.1/32'.format(switch_name[-2::]))

            for route in routes_to_check:
                if route not in all_routes:
                    RESULT_STATUS = False
                    failure_summary += 'On switch {} '.format(switch_name)
                    failure_summary += 'bgp route for network {} '.format(route)
                    failure_summary += 'is not showing up '
                    failure_summary += 'in the output of {} for {}th retry\n'.format(cmd, retries)

            for route in all_routes.splitlines():
                route = route.strip()
                for network in routes_to_check:
                    if network in route:
                        if '20/' or '200/' in route:
                            pass
                        else:
                            RESULT_STATUS = False
                            failure_summary += 'On switch {} '.format(switch_name)
                            failure_summary += 'administrative value is not present '
                            failure_summary += 'in the bgp route {} for {}th retry.\n'.format(route, retries)

                        # if not route.startswith('B>*'):
                        #     RESULT_STATUS = False
                        #     failure_summary += 'On switch {} '.format(switch_name)
                        #     failure_summary += 'bgp route {} '.format(route)
                        #     failure_summary += 'does not start with B>*\n'
        else:
            RESULT_STATUS = False
            failure_summary += 'On switch {} '.format(switch_name)
            failure_summary += 'administrative distance cannot be verified since '
            failure_summary += 'output of command {} is None for {}th retry.'.format(cmd, retries)

        if not RESULT_STATUS:
            retries -= 1
            time.sleep(delay)
        else:
            break

    # Store the failure summary in hash
    HASH_DICT['result.detail'] = failure_summary

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            package_name=dict(required=False, type='str'),
            spine_list=dict(required=False, type='list', default=[]),
            leaf_list=dict(required=False, type='list', default=[]),
            delay=dict(required=False, type='int', default=15),
            retries=dict(required=False, type='int', default=50),
            dry_run_mode=dict(required=False, type='bool', default=False),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS

    verify_gobgp_administrative_distance(module)

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

