#!/usr/bin/python
""" Test/Verify BIRD PEERING IF DOWN """

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
module: test_bird_peering_if_down
author: Platina Systems
short_description: Module to test and verify bird configuration.
description:
    Module to test and verify bird configurations and log the same.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    config_file:
      description:
        - BGP config which have been added into /etc/bird/bird.conf.
      required: False
      type: str
    package_name:
      description:
        - Name of the package installed (e.g. quagga/frr/bird).
      required: False
      type: str
    leaf_list:
      description:
        - List of all leaf switches.
      required: False
      type: list
      default: []
    eth_list:
      description:
        - Comma separated string of eth interfaces to bring down/up.
      required: False
      type: str
    check_ping:
      description:
        - Flag to indicate if ping should be tested or not.
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
- name: Verify bird peering if down
  test_bird_peering_if_down:
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


def verify_ping(module):
    """
    Method to verify bgp neighbor relationship.
    :param module: The Ansible module to fetch input parameters.
    :return: Failure summary if any.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    leaf_list = module.params['leaf_list']
    config_file = module.params['config_file'].splitlines()
    self_ip = '192.168.{}.1'.format(switch_name[-2::])
    neighbor = []

    for line in config_file:
        line = line.strip()
        if 'neighbor' in line and 'as' in line:
            config = line.split()
            neighbor.append(config[1])

    for leaf in leaf_list:
        if switch_name not in leaf:
            ip = '192.168.{}.1'.format(leaf[-2::])
            neighbor.append(ip)

    for ip in neighbor:
        packet_count = '3'
        ping_cmd = 'ping -w 3 -c {} -I {} {}'.format(packet_count,
                                                     self_ip, ip)
        ping_out = execute_commands(module, ping_cmd)
        if '100% packet loss'.format(packet_count) in ping_out:
            RESULT_STATUS = False
            failure_summary += 'From switch {} '.format(
                switch_name)
            failure_summary += 'neighbor ip {} '.format(ip)
            failure_summary += 'is not getting pinged\n'

    return failure_summary


def check_bgp_neighbors(module, neighbor_ips, neighbor_as):
    """
    Method to verify bgp neighbor relationship.
    :param module: The Ansible module to fetch input parameters.
    :param neighbor_ips: List of neighbor ips.
    :param neighbor_as: List of remote as of neighbors.
    :return: Failure summary if any.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    switch_name = module.params['switch_name']
    check_ping = module.params['check_ping']
    leaf_list = module.params['leaf_list']
    is_leaf = True if switch_name in leaf_list else False

    for ip in neighbor_ips:
        index = neighbor_ips.index(ip)
        as_value = neighbor_as[index]
	time.sleep(module.params['delay'])
        cmd = "birdc 'show protocols all bgp{}'".format(index + 1)
        bgp_out = execute_commands(module, cmd)

        if bgp_out:
            if ip not in bgp_out and as_value not in bgp_out:
                RESULT_STATUS = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'bgp neighbor {} info '.format(ip)
                failure_summary += 'is not present in the output of '
                failure_summary += 'command {}\n'.format(cmd)

            if 'Established' not in bgp_out:
                RESULT_STATUS = False
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'bgp state of neighbor {} '.format(ip)
                failure_summary += 'is not Established in the output of '
                failure_summary += 'command {}\n'.format(cmd)

        else:
            RESULT_STATUS = False
            failure_summary += 'On switch {} '.format(switch_name)
            failure_summary += 'result cannot be verified since '
            failure_summary += 'output of command {} is None'.format(cmd)

    if check_ping and is_leaf:
        failure_summary += verify_ping(module)

    return failure_summary


def change_interface_state(module, eth_list, state):
    """
    Method to bring up/down eth interfaces.
    :param module: The Ansible module to fetch input parameters.
    :param eth_list: List of eth interfaces.
    :param state: State of the interface, either up or down.
    """
    # Bring down few eth interfaces on only leaf switches
    for eth in eth_list:
        eth = eth.strip()
        cmd = 'ifconfig xeth{} {}'.format(eth, state)
        execute_commands(module, cmd)


def verify_bird_peering_if_down(module):
    """
    Method to verify bird peering if down.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    failure_summary = ''
    neighbor_ips = []
    neighbor_as = []
    switch_name = module.params['switch_name']
    package_name = module.params['package_name']
    check_ping = module.params['check_ping']
    config_file = module.params['config_file'].splitlines()
    eth_list = module.params['eth_list'].split(',')
    leaf_list = module.params['leaf_list']
    is_leaf = True if switch_name in leaf_list else False

    # Get the current/running configurations
    execute_commands(module, 'cat /etc/bird/bird.conf')

    # Restart and check package status
    # execute_commands(module, 'service {} restart'.format(package_name))
    execute_commands(module, 'service {} status'.format(package_name))

    for line in config_file:
        line = line.strip()
        if 'neighbor' in line and 'as' in line:
            config = line.split()
            neighbor_ips.append(config[1])
            neighbor_as.append(config[3])

    # Verify bgp neighbor relationship
    failure_summary += check_bgp_neighbors(module, neighbor_ips, neighbor_as)

    # Bring down the interfaces of leaf switches
    if is_leaf:
        change_interface_state(module, eth_list, 'down')

    # Wait for 160 seconds
    if not check_ping:
        time.sleep(160)

    # Verify bgp neighbor relationship
    failure_summary += check_bgp_neighbors(module, neighbor_ips, neighbor_as)

    # Bring up the interfaces of leaf switches
    if is_leaf:
        change_interface_state(module, eth_list, 'up')

    # Wait for 40 seconds
    if not check_ping:
        time.sleep(40)

    # Verify bgp neighbor relationship
    failure_summary += check_bgp_neighbors(module, neighbor_ips, neighbor_as)

    HASH_DICT['result.detail'] = failure_summary

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            config_file=dict(required=False, type='str'),
            delay=dict(required=False, type='int'),
            retries=dict(required=False, type='int'),
            dry_run_mode=dict(required=False, type='bool', default=False),
            package_name=dict(required=False, type='str'),
            leaf_list=dict(required=False, type='list', default=[]),
            eth_list=dict(required=False, type='str'),
            check_ping=dict(required=False, type='bool', default=False),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS

    if module.params['dry_run_mode']:
        package_name = module.params['package_name']
        cmds_list = []
        is_leaf = True if module.params["switch_name"] in module.params["leaf_list"] else False
        execute_commands(module, 'cat /etc/bird/bird.conf')
        execute_commands(module, 'service {} status'.format(package_name))

        for index in (1, 2):
            execute_commands(module, "birdc 'show protocols all bgp{}'".format(index))
        self_ip = '192.168.{}.1'.format(module.params["switch_name"][-2::])
        if module.params['check_ping'] and is_leaf:
            neighbor = list()
            for leaf in module.params["leaf_list"]:
                if module.params["switch_name"] not in leaf:
                    ip = '192.168.{}.1'.format(leaf[-2::])
                    neighbor.append(ip)

            for ip in neighbor:
                packet_count = '3'
                execute_commands(module, 'ping -w 3 -c {} -I {} {}'.format(packet_count, self_ip, ip))

        if is_leaf:
            change_interface_state(module, module.params['eth_list'].split(','), 'down')

        if not module.params["check_ping"]:
            time.sleep(160)

        for index in (0,1):
            execute_commands(module, "birdc 'show protocols all bgp{}'".format(index + 1))

        if module.params['check_ping'] and is_leaf:
            neighbor = list()
            for leaf in module.params["leaf_list"]:
                if module.params["switch_name"] not in leaf:
                    ip = '192.168.{}.1'.format(leaf[-2::])
                    neighbor.append(ip)

            for ip in neighbor:
                packet_count = '3'
                execute_commands(module, 'ping -w 3 -c {} -I {} {}'.format(packet_count, self_ip, ip))

        if is_leaf:
            change_interface_state(module, module.params['eth_list'].split(','), 'up')

        if not module.params["check_ping"]:
            time.sleep(40)

        for index in (0, 1):
            execute_commands(module, "birdc 'show protocols all bgp{}'".format(index + 1))

        execute_commands(module, 'goes status')

        for key, value in HASH_DICT.iteritems():
            cmds_list.append(key)
        # Exit the module and return the required JSON.
        module.exit_json(
            cmds=cmds_list
        )
    else:

        verify_bird_peering_if_down(module)

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

