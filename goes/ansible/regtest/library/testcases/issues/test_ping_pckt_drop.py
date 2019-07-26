#!/usr/bin/python
""" Verify Ping Packet Drops """

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
module: test_ping_pckt_drop
author: Platina Systems
short_description: Module to test ping packet drops when traffic is passing.
description:
    Module to test ping packet drops when traffic is passing.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    eth:
      description:
        - Name of the interface on which ping is to be tested.
      required: False
      type: str
    target_switch:
      description:
        - Name of spine switch on which iperf server is to be created.
      required: False
      type: list
      default: []
    iperf:
      description:
        - Flag to indicate if iperf to be used.
      required: False
      type: bool
      default: True
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
- name: Initiate iperf client and verify packet drops
  test_ping_pckt_drop:
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
import time

def verify_traffic(module):
    """
    Method to verify iperf traffic.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS, HASH_DICT
    port = 4000
    failure_summary = ''
    switch_name = module.params['switch_name']
    target_switch = module.params['target_switch']
    eth = module.params['eth']
    iperf = module.params['iperf']

    is_target = True if switch_name in target_switch else False

    neighbor_ip = '10.0.1.{}'.format(target_switch[-2::])

    if is_target and iperf:
        cmd = 'iperf -s -p {}'.format(port)
        execute_commands(module, cmd)

    elif iperf:
        # Initiate iperf client and verify traffic
        traffic_cmd = 'iperf -c {} -p {} -t 100 -P 2'.format(neighbor_ip, port)
        execute_commands(module, traffic_cmd)

        # Verify ping
        ping_cmd = 'ping -c 10 {}'.format(neighbor_ip)
        ping_out = execute_commands(module, ping_cmd)
        if '0% packet loss' not in ping_out:
            RESULT_STATUS = False
            failure_summary += 'On switch {}, '.format(switch_name)
            failure_summary += 'packet loss is observed.\n'

    else:
	time.sleep(5)
        cmd = 'goes hget platina-mk1 link'
        link_status = execute_commands(module, cmd).splitlines()
	#print(link_status)
        for line in link_status:
            if 'xeth{}.link'.format(eth) in line and 'false' in line:
	#	print("!!!!!!!!! {}".format(line))
                RESULT_STATUS = False
                failure_summary += 'On switch {}, '.format(switch_name)
                failure_summary += 'link status of xeth{} is not true.\n'.format(eth)

        cmd = 'ping -s 2000 10.0.1.{} -c 5'.format(target_switch[-2:][0][-2:])
        ping_out = execute_commands(module, cmd)
        if '0% packet loss' not in ping_out:
            RESULT_STATUS = False
            failure_summary += 'On switch {}, '.format(switch_name)
            failure_summary += 'packet loss is observed.\n'

    if iperf:
        # verify mmu_multicast_tx_cos0 packet drops
        cmd = 'timeout 10 watch -n 1 "goes hgetall platina-mk1"'
        out = execute_commands(module, cmd).splitlines()

        for line in out:
            if 'grep eth-{}-1'.format(eth) in line and 'mmu_multicast_tx_cos0' in line:
                line = line.strip()
                drops = line.split()[1]
                if drops == '0':
                    RESULT_STATUS = False
                    failure_summary += 'On switch {}, '.format(switch_name)
                    failure_summary += 'mmu_multicast_tx_cos0 packet drops are observed.\n'

    HASH_DICT['result.detail'] = failure_summary

    # Get the GOES status info
    execute_commands(module, 'goes status')


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            eth=dict(required=False, type='str'),
            target_switch=dict(required=False, type='list', default=[]),
            iperf=dict(required=False, type='bool', default='True'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global RESULT_STATUS, HASH_DICT

    verify_traffic(module)

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


