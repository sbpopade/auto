#!/usr/bin/python
""" Test/Verify ISIS Basic """

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
module: test_isis_basic
author: Platina Systems
short_description: Module to test and verify isis config.
description:
    Module to test and verify isis configurations and log the same.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
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
    check_neighbors:
      description:
        - Flag to indicate if isis neighbors to check or not.
      required: False
      type: bool
      default: False
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
- name: Verify bgp peering loopback
  test_bgp_peering_loopback:
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


def verify_isis_neighbors(module):
    """
    Method to verify isis config.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    package_name = module.params['package_name']
    spine_list = module.params['spine_list']
    leaf_list = module.params['leaf_list']
    is_spine = True if switch_name in spine_list else False

    # Get the current/running configurations
    execute_commands(module, "vtysh -c 'sh running-config'")

    # Restart and check package status
    execute_commands(module, 'service {} restart'.format(package_name))
    execute_commands(module, 'service {} status'.format(package_name))

    retries = 1
    found = False
    retries_summary = ''

    while retries != 6 and not found:
        # Wait 50 secs(max) for routes to become reachable
        time.sleep(10)
        summary = ''

        # Get isis neighbor info
        cmd = "vtysh -c 'sh isis neighbor'"
        isis_out = execute_commands(module, cmd)

        if not isis_out:
            summary += 'On Switch {} '.format(switch_name)
            summary += 'isis neighbors cannot be verified since '
            summary += 'output of command {} is None\n'.format(cmd)
        else:
            if is_spine:
                for leaf in leaf_list:
                    if isis_out.count(leaf) != 2:
                        summary += 'On switch {} '.format(switch_name)
                        summary += 'two neighbors for {} is not '.format(
                            leaf)
                        summary += 'present in the output of {}\n'.format(
                            cmd)
            else:
                for spine in spine_list:
                    if isis_out.count(spine) != 2:
                        summary += 'On switch {} '.format(switch_name)
                        summary += 'two neighbors for {} is not '.format(
                            spine)
                        summary += 'present in the output of {}\n'.format(
                            cmd)

            if isis_out.count('Up') != 4:
                summary += 'On switch {} '.format(switch_name)
                summary += 'isis neighbors state is not Up\n'

        if not summary:
            found = True
            summary = 'No. of retries {}'.format(retries)
        else:
            retries += 1

    if not found:
        RESULT_STATUS = False
        failure_summary += summary
        retries_summary += 'No. of retries {} approx {} sec(Get isis neighbor info)\n'.format(retries,
                                                                                            retries * 10)
    else:
        retries_summary += 'No. of retries {} approx {} sec(Get isis neighbor info)\n'.format(retries,
                                                                                            retries * 10)
    retries = 1
    found = False

    while retries != 6 and not found:
        # Wait 50 secs(max) for routes to become reachable
        time.sleep(10)
        summary = ''

        # Check and verify neighbor routes
        if module.params['check_neighbors']:
            cmd = "vtysh -c 'sh ip route'"
            all_routes = execute_commands(module, cmd)
            route_count = 0

            for route in all_routes.splitlines():
                if route.startswith('I'):
                    route_count += 1
                    if '115' not in route:
                        summary += 'On switch {} '.format(switch_name)
                        summary += 'administrative value 115 is not present'
                        summary += ' in isis route {}\n'.format(route)

            if route_count < 4:
                summary += 'On switch {} '.format(switch_name)
                summary += 'output of {} '.format(cmd)
                summary += 'is not displaying required isis routes\n'

        if not summary:
            found = True
            summary = 'No. of retries {}'.format(retries)
        else:
            retries += 1

    if not found:
        RESULT_STATUS = False
        failure_summary += summary
        retries_summary += 'No. of retries {} approx {} sec(Check and verify neighbor routes)\n'.format(retries,
                                                                                            retries * 10)
    else:
        retries_summary += 'No. of retries {} approx {} sec(Check and verify neighbor routes)\n'.format(retries,
                                                                                            retries * 10)

    HASH_DICT['retries'] = retries_summary
    HASH_DICT['result.detail'] = failure_summary

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            spine_list=dict(required=False, type='list', default=[]),
            leaf_list=dict(required=False, type='list', default=[]),
            check_neighbors=dict(required=False, type='bool', default=False),
            package_name=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS

    verify_isis_neighbors(module)

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

