#!/usr/bin/python
""" Test/Verify BGP Route Advertise """

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
module: test_bgp_route_advertise
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
    package_name:
      description:
        - Name of the package installed (e.g. quagga/frr/bird).
      required: False
      type: str
    is_ibgp:
      description:
        - Flag to indicate if ibgp config needs to be verified or ebgp.
      required: False
      type: bool
      default: False
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
- name: Verify bgp peering route advertise
  test_bgp_route_advertise:
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

def verify_route_advertise(module):
    routes_to_check = []
    switch_name = module.params['switch_name']
    is_ibgp = module.params['is_ibgp']
    spine_list = module.params['spine_list']
    leaf_list = module.params['leaf_list']

    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    RESULT_STATUS1 = True
    cmd = "vtysh -c 'sh ip bgp'"
    bgp_out = execute_commands(module, cmd)

    if bgp_out:
        if is_ibgp:
            is_spine = True if switch_name in spine_list else False
            switches_list = leaf_list if is_spine else spine_list
            switches_list.append(switch_name)
            if is_spine:
		aleaf_list = leaf_list[:]
		aleaf_list.remove(switch_name)
		switches_list.remove(aleaf_list[0])
        else:
            switches_list = spine_list + leaf_list

        for switch in switches_list:
            routes_to_check.append('192.168.{}.1/32'.format(switch[-2::]))

#        if switch_name in leaf_list:
#		leaf_list.remove(switch_name)
#		print(switch_name)
#		print(leaf_list)
#		print('192.168.{}.1/32'.format(leaf_list[0][-2::]))
#		print(routes_to_check)
#		raise("Intentional")
		#routes_to_check.remove('192.168.{}.1/32'.format(leaf_list[0][-2::]))

        for route in routes_to_check:
            if route not in bgp_out:
                RESULT_STATUS1 = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'bgp route for network {} '.format(route)
                failure_summary += 'is not showing up '
                failure_summary += 'in the output of {}\n'.format(cmd)
    else:
        RESULT_STATUS1 = False
        failure_summary += 'On switch {} '.format(switch_name)
        failure_summary += 'bgp routes cannot be verified '
        failure_summary += 'because output of command {} '.format(cmd)
        failure_summary += 'is None'

    alist = [True if RESULT_STATUS1 else False]
    alist.append(failure_summary)
    return alist

def verify_ping(module):
    switch_name = module.params['switch_name']

    is_ping = module.params['is_ping']
    leaf_list1 = module.params['leaf_list'][:]
    spine_list1 = module.params['spine_list'][:]
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    RESULT_STATUS1 = True
    if is_ping:
        packet_count = 5
        if switch_name in leaf_list1:
                p_list = leaf_list1
        elif switch_name in spine_list1:
                p_list = spine_list1

        p_list.remove(switch_name)
        cmd = "ping -c {} -I 192.168.{}.1 192.168.{}.1".format(packet_count, switch_name[-2:], p_list[0][-2:])

        ping_out = execute_commands(module, cmd)

        if '100% packet loss' in ping_out:
            RESULT_STATUS1 = False
            failure_summary += 'Ping from switch {} to {}'.format(switch_name, p_list[0])
            failure_summary += ' for {} packets'.format(packet_count)
            failure_summary += ' are not received in the output of '
            failure_summary += 'command {}\n'.format(cmd)

    alist = [True if RESULT_STATUS1 else False]
    alist.append(failure_summary)
    return alist

def verify_bgp_route_advertise(module):
    """
    Method to verify bgp route advertise.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    package_name = module.params['package_name']
    delay = module.params['delay']
    retries = module.params['retries']

    # Get the current/running configurations
    execute_commands(module, "vtysh -c 'sh running-config'")

    # Restart and check package status
    # execute_commands(module, 'service {} restart'.format(package_name))
    execute_commands(module, 'service {} status'.format(package_name))

    # Get all bgp routes
    retry = retries - 1
    found_list = [False, False]
    while(retry):
        if not found_list[0]:
            if verify_route_advertise(module)[0]:
                found_list[0] = True

        if not found_list[1]:
            if verify_ping(module)[0]:
                found_list[1] = True

        if all(found_list):
            break
        else:
            time.sleep(delay)
            retry -= 1

    # Store the failure summary in hash
    RESULT_STATUS, HASH_DICT['result.detail'] = all([verify_route_advertise(module)[0], verify_ping(module)[0]]), verify_route_advertise(module)[1] + verify_ping(module)[1]

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            spine_list=dict(required=False, type='list', default=[]),
            leaf_list=dict(required=False, type='list', default=[]),
            is_ping=dict(required=False, type='bool'),
	        package_name=dict(required=False, type='str'),
            is_ibgp=dict(required=False, type='bool', default=False),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
            delay=dict(required=False, type='int', default=10),
            retries=dict(required=False, type='int', default=6),
            dry_run_mode=dict(required=False, type='bool', default=False),
        )
    )

    global HASH_DICT, RESULT_STATUS
    if module.params['dry_run_mode']:
        switch_name = module.params['switch_name']
        is_ping = module.params['is_ping']
        leaf_list1 = module.params['leaf_list'][:]
        spine_list1 = module.params['spine_list'][:]
        package_name = module.params['package_name']
        cmds_list = []

        execute_commands(module, "vtysh -c 'sh running-config'")

        # Restart and check package status
        execute_commands(module, 'service {} restart'.format(package_name))
        execute_commands(module, 'service {} status'.format(package_name))
        cmd = "vtysh -c 'sh ip bgp'"
        execute_commands(module, cmd)
        if is_ping:
            packet_count = 5
            if switch_name in leaf_list1:
                p_list = leaf_list1
            elif switch_name in spine_list1:
                p_list = spine_list1

            p_list.remove(switch_name)
            cmd = "ping -c {} -I 192.168.{}.1 192.168.{}.1".format(packet_count, switch_name[-2:], p_list[0][-2:])

            execute_commands(module, cmd)
        execute_commands(module, 'goes status')

        for key, value in HASH_DICT.iteritems():
            cmds_list.append(key)

        # Exit the module and return the required JSON.
        module.exit_json(
            cmds=cmds_list
        )
    else:
        verify_bgp_route_advertise(module)

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

