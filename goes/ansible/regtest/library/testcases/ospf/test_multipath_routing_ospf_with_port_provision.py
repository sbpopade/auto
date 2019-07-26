#!/usr/bin/python
""" Test/Verify OSPF Config """

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
module: test_ospf_multipath
author: Platina Systems
short_description: Module to test and verify ospf configurations.
description:
    Module to test and verify ospf configurations and log the same.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    eth_list:
      description:
        - Comma separated string of eth interfaces to bring down/up.
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
    interface:
      description:
        - Name of the interface which has to propagate(up/down).
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
- name: Verify ospf config
  test_ospf_basic:
    switch_name: "{{ inventory_hostname }}"
    spine_list: "{{ groups['spine'] }}"
    spine_ips: "{{ groups['spine'] | map('extract', hostvars, ['ansible_ssh_host']) | join(',') }}"
    leaf_ips: "{{ groups['leaf'] | map('extract', hostvars, ['ansible_ssh_host']) | join(',') }}"
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


def get_config(module):
    """
    Method to get running config and package status.
    :param module: The Ansible module to fetch input parameters.
    """
    package_name = module.params['package_name']

    # Get the current/running configurations
    execute_commands(module, "vtysh -c 'sh running-config'")

    # Check package status
    execute_commands(module, 'service {} status'.format(package_name))


def verify_dummy_ping(module):
    """
    Method to verify ping for dummy interface config.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    leaf_list = module.params['leaf_list']
    stage = module.params['stage']

    leaf_list.remove(switch_name)
    nei_ip = '192.168.{}.1'.format(leaf_list[0][-2::])
    cmd = 'ping -c 5 -I 192.168.{}.1 {}'.format(switch_name[-2::], nei_ip)
    out = execute_commands(module, cmd)
    if '100% packet loss' in out:
        RESULT_STATUS = False
        failure_summary += 'From switch {} '.format(switch_name)
        failure_summary += 'neighbor ip {} '.format(nei_ip)
        failure_summary += 'is not getting pinged {}\n'.format(stage)

    return failure_summary


def verify_fib(module):
    """
    Method to verify if fib entries reflects.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    leaf_list = module.params['leaf_list']
    spine_list = module.params['spine_list']
    stage = module.params['stage']
    eth_list = module.params['eth_list'].split(',')
    interface = module.params['interface']

    routes = []
    for switch in leaf_list + spine_list:
        ip = '192.168.{}.1'.format(switch[-2::])
        routes.append(ip)

    cmd = 'goes vnet show ip fib'
    out = execute_commands(module, cmd)

    for route in routes:
        if route not in out:
            RESULT_STATUS = False
            failure_summary += 'On Switch {} '.format(switch_name)
            failure_summary += 'fib entry for {} cannot be verified in the '.format(route)
            failure_summary += 'output of command {} {}\n'.format(cmd, stage)

    if interface:
       for inter in interface:
         if stage in "after bringing interface up":
            if 'rewrite xeth{}'.format(inter) not in out:
                RESULT_STATUS = False
                failure_summary += 'On Switch {} '.format(switch_name)
                failure_summary += 'ip route for xeth{} is not there even {} '.format(inter, stage)
                failure_summary += 'in the output of command {}\n'.format(cmd)
         else:
            if 'xeth{} weight 1'.format(inter) in out:
                RESULT_STATUS = False
                failure_summary += 'On Switch {} '.format(switch_name)
                failure_summary += 'ip route for xeth{} is still there even {} '.format(inter, stage)
                failure_summary += 'in the output of command {}\n'.format(cmd)
    else:
      for eth in eth_list:
        if 'rewrite xeth{}'.format(eth) not in out:
            RESULT_STATUS = False
            failure_summary += 'On Switch {} '.format(switch_name)
            failure_summary += 'fib entry for xeth{} cannot be verified in the '.format(eth)
            failure_summary += 'in the output of command {} {}\n'.format(cmd, stage)

    return failure_summary


def verify_ip_route(module):
    """
    Method to verify if fib entries reflects.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    leaf_list = module.params['leaf_list']
    spine_list = module.params['spine_list']
    eth_list = module.params['eth_list'].split(',')
    interface = module.params['interface']
    stage = module.params['stage']

    routes = []
    for switch in leaf_list + spine_list:
        if switch not in switch_name:
            ip = '192.168.{}.1'.format(switch[-2::])
            routes.append(ip)

    cmd = 'ip route'
    out = execute_commands(module, cmd)

    for route in routes:
        if route not in out:
            RESULT_STATUS = False
            failure_summary += 'On Switch {} '.format(switch_name)
            failure_summary += '{} route cannot be found in the '.format(route)
            failure_summary += 'output of command {} {}\n'.format(cmd, stage)

    if interface:
       for inter in interface:
         if stage in "after bringing interface up":
            if 'xeth{} weight 1'.format(inter) not in out:
                RESULT_STATUS = False
                failure_summary += 'On Switch {} '.format(switch_name)
                failure_summary += 'ip route for xeth{} is not there even {} '.format(inter, stage)
                failure_summary += 'in the output of command {}\n'.format(cmd)
         else:
            if 'xeth{} weight 1'.format(inter) in out:
            	RESULT_STATUS = False
            	failure_summary += 'On Switch {} '.format(switch_name)
            	failure_summary += 'ip route for xeth{} is still there even {} '.format(inter, stage)
            	failure_summary += 'in the output of command {}\n'.format(cmd)

    else:
      for eth in eth_list:
        if interface not in eth and 'dev xeth{} weight 1'.format(eth) not in out:
            RESULT_STATUS = False
            failure_summary += 'On Switch {} '.format(switch_name)
            failure_summary += 'ip route for xeth{} cannot be verified '.format(interface)
            failure_summary += 'in the output of command {} {}\n'.format(cmd, stage)

    return failure_summary


def verify_ospf_neighbors(module):
    """
    Method to verify if ospf neighbor relationship got established or not.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    stage = module.params['stage']
    interface = module.params['interface']
    eth_list = module.params['eth_list'].split(',')

    cmd = "vtysh -c 'show ip ospf neighbor'"
    out = execute_commands(module, cmd)

    for eth in eth_list:
        if out is None or 'error' in out:
            RESULT_STATUS = False
            failure_summary += 'On Switch {} '.format(switch_name)
            failure_summary += 'ospf neighbors cannot be verified since '
            failure_summary += 'output of command {} is None {}\n'.format(cmd, stage)
        else:
            if interface and interface in eth and 'xeth{}'.format(eth) in out:
                RESULT_STATUS = False
                failure_summary += 'On Switch {} '.format(switch_name)
                failure_summary += 'ospf neighbor for interface xeth{} is still there '.format(interface)
                failure_summary += 'even {} in the '.format(stage)
                failure_summary += 'output of command {}\n'.format(cmd)

            elif interface not in eth and 'xeth{}'.format(eth) not in out:
                RESULT_STATUS = False
                failure_summary += 'On Switch {} '.format(switch_name)
                failure_summary += 'ospf neighbors are not present '
                failure_summary += 'in the output of command {} {}\n'.format(cmd, stage)

    return failure_summary


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            eth_list=dict(required=False, type='str'),
            spine_list=dict(required=False, type='list', default=[]),
            leaf_list=dict(required=False, type='list', default=[]),
            interface=dict(required=False, type='str', default=''),
            package_name=dict(required=False, type='str'),
            stage=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
            is_ping=dict(required=False, type='bool', default=False),
        )
    )

    global HASH_DICT, RESULT_STATUS
    failure_summary = ''
    switch_name = module.params['switch_name']
    leaf_list = module.params['leaf_list']
    stage = module.params['stage']

    is_leaf = True if switch_name in leaf_list else False

    # Get running configs and package status initially
    if 'initial stage' in stage:
        get_config(module)

    # Verify ospf neighbors
    failure_summary += verify_ospf_neighbors(module)
    failure_summary += verify_fib(module)
    failure_summary += verify_ip_route(module)
    if is_leaf and module.params['is_ping']:
        failure_summary += verify_dummy_ping(module)

    HASH_DICT['result.detail'] = failure_summary

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

