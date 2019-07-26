#!/usr/bin/python
""" Test/Verify BGP Convergence """

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

import time

from ansible.module_utils.basic import AnsibleModule

DOCUMENTATION = """
---
module: test_bgp_convergence
author: Platina Systems
short_description: Module to test and verify bgp convergence.
description:
    Module to test and verify bgp configurations and log the same.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    converge_switch:
      description:
        - Name of the switch on which network has been converged.
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
- name: Verify bgp convergence
  test_bgp_convergence:
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

def verify_ip_routes(module):

    switch_name = module.params['switch_name']
    converge_switch = module.params['converge_switch']
    converge_switch_config = module.params['converge_switch_config']

    global RESULT_STATUS
    RESULT_STATUS1 = True
    failure_summary = ''
    cmd = "vtysh -c 'sh ip route'"
    routes_out = execute_commands(module, cmd)

    if routes_out:
        route = 'B>* 192.168.{}.1'.format(converge_switch[-2::])

        if route in routes_out and converge_switch_config == 'absent':
            RESULT_STATUS1 = False
            failure_summary += 'On Switch {} bgp route '.format(switch_name)
            failure_summary += 'for network {} is present '.format(route)
            failure_summary += 'in the output of command {} '.format(cmd)
            failure_summary += 'even after removing this network\n'

        elif route not in routes_out and converge_switch_config == 'present':
            RESULT_STATUS1 = False
            failure_summary += 'On Switch {} bgp route '.format(switch_name)
            failure_summary += 'for network {} is absent '.format(route)
            failure_summary += 'in the output of command {} '.format(cmd)
            failure_summary += 'even after presence of this network\n'

    else:
        RESULT_STATUS1 = False
        failure_summary += 'On switch {} '.format(switch_name)
        failure_summary += 'bgp convergence cannot be verified '
        failure_summary += 'because output of command {} is None'.format(cmd)

    alist = [True if RESULT_STATUS1 else False]
    alist.append(failure_summary)
    return alist

def verify_ping(module):

    global RESULT_STATUS
    RESULT_STATUS1 = True
    failure_summary = ''
    switch_name = module.params['switch_name']
    converge_switch = module.params['converge_switch']
    converge_switch_config = module.params['converge_switch_config']
    is_ping = module.params['is_ping']

    if is_ping and converge_switch_config == 'present':
        packet_count = 5
        cmd = "ping -c {} -I 192.168.{}.1 192.168.{}.1".format(packet_count, switch_name[-2:], converge_switch[-2:])

        ping_out = execute_commands(module, cmd)

        if '100% packet loss' in ping_out:
            RESULT_STATUS1 = False
            failure_summary += 'Ping from switch {} to {}'.format(switch_name, converge_switch)
            failure_summary += ' for {} packets'.format(packet_count)
            failure_summary += ' are not received in the output of '
            failure_summary += 'command {}\n'.format(cmd)

    alist = [True if RESULT_STATUS1 else False]
    alist.append(failure_summary)
    return alist


def verify_bgp_quagga_convergence(module):
    """
    Method to verify bgp quagga convergence.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    switch_name = module.params['switch_name']
    package_name = module.params['package_name']
    converge_switch = module.params['converge_switch']
    delay = module.params['delay']
    retries = module.params['retries']

    # Get the current/running configurations
    execute_commands(module, "vtysh -c 'sh running-config'")

    # Check package status
    execute_commands(module, 'service {} status'.format(package_name))

    if switch_name != converge_switch:
        # Get all ip routes
        retry = retries - 1
        found_list = [False, False]
        while(retry):
            if not found_list[0]:
                if verify_ip_routes(module)[0]:
                    found_list[0] = True

            if not found_list[1]:
                if verify_ping(module)[0]:
                    found_list[1] = True

            if all(found_list):
                break
            else:
                time.sleep(delay)
                retry -= 1

    RESULT_STATUS, HASH_DICT['result.detail'] = all([verify_ip_routes(module)[0], verify_ping(module)[0]]), verify_ip_routes(module)[1] + verify_ping(module)[1]

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
	        spine_list=dict(required=False, type='list', default=[]),
            leaf_list=dict(required=False, type='list', default=[]),
            delay=dict(required=False, type='int', default=10),
            retries=dict(required=False, type='int', default=6),
            dry_run_mode=dict(required=False, type='bool', default=False),
            is_ping=dict(required=False, type='bool'),
            converge_switch=dict(required=False, type='str'),
            converge_switch_config=dict(required=False, type='str'),
            package_name=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS

    if module.params['dry_run_mode']:
        cmds_list = []

        switch_name = module.params['switch_name']
        package_name = module.params['package_name']
        converge_switch = module.params['converge_switch']
        converge_switch_config = module.params['converge_switch_config']
        spine_list = module.params['spine_list']
        leaf_list = module.params['leaf_list']
        is_ping = module.params['is_ping']

        # Get the current/running configurations
        execute_commands(module, "vtysh -c 'sh running-config'")

        # Check package status
        execute_commands(module, 'service {} status'.format(package_name))

        if switch_name != converge_switch:
            # Get all ip routes
            cmd = "vtysh -c 'sh ip route'"
            execute_commands(module, cmd)

            if is_ping and converge_switch_config == 'present':
                packet_count = 5
                cmd = "ping -c {} -I 192.168.{}.1 192.168.{}.1".format(packet_count, switch_name[-2:],
                                                                       converge_switch[-2:])
                execute_commands(module, cmd)

        # Get the GOES status info
        execute_commands(module, 'goes status')

        for key, value in HASH_DICT.iteritems():
            cmds_list.append(key)

        # Exit the module and return the required JSON.
        module.exit_json(
            cmds=cmds_list
        )
    else:

        verify_bgp_quagga_convergence(module)

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

