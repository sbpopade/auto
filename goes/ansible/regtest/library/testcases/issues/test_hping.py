#!/usr/bin/python
""" Test hping """

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
module: test_hping
author: Platina Systems
short_description: Module to restart goes and check the status.
description:
    Module to restart goes for given number of times and check the status.
options:
    switch_name:
      description:
        - Name of the switch from which ping will be checked.
      required: False
      type: str
    target_name:
      description:
        - Name of the switch to which ping will be checked.
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
- name: Restart goes 1500 times and check it's status
  restart_and_check_goes_status:
    switch_name1: "{{ inventory_hostname }}"
    switch_name2: "{{ groups['spine'][0] }}"
    hash_name: "{{ hostvars['server_emulator']['hash_name'] }}"
    log_dir_path: "{{ log_dir_path }}"
"""

RETURN = """
changed:
  description: Boolean flag to indicate if any changes were made by this module.
  returned: always
  type: bool
hash_dict:
  description: Dictionary containing key value pairs to store in hash.
  returned: always
  type: dict
log_file_path:
  description: Path to the log file on this switch.
  returned: always
  type: str
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


def verify_ping(module):
	global RESULT_STATUS, failure_summary
	failure_summary = ''
        switch_name = module.params['switch_name']
	switch_name1 = module.params['switch_name1']
	switch_name2 = module.params['switch_name2']
	spine_list = module.params['spine_list']

	if switch_name in spine_list:
		ip1 = "10.0.5.{}".format(switch_name1[-2::])
		ip2 = "10.0.5.{}".format(switch_name2[-2::])
	else:
		ip2 = "10.0.5.{}".format(switch_name1[-2::])
		ip1 = "10.0.5.{}".format(switch_name2[-2::])

	cmd1 = "ping -c 5 -I {} {}".format(ip1, ip2)
	cmd2 = "timeout 5 hping3 -c 5 --icmp --faster -t 1 {}".format(ip2)

        out1 = execute_commands(module, cmd1)
	if "100% packet loss" in out1:
		RESULT_STATUS = False
		failure_summary += "ping from {} to ".format(ip1)
		failure_summary += "{} is not working.\n".format(ip2)

	out2 = execute_commands(module, cmd2)
        #time.sleep(5)
	if len(out2.splitlines()) < 10:
		RESULT_STATUS = False
		failure_summary += "hping from {} to ".format(switch_name[-2::])
		failure_summary += "{} is not working.\n".format(ip2)

	HASH_DICT['result.detail'] = failure_summary


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
 	    switch_name=dict(required=False, type='str'),
            switch_name1=dict(required=False, type='str'),
	    switch_name2=dict(required=False, type='str'),
	    spine_list=dict(required=False, type='list'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS
    verify_ping(module)
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
        log_file_path=log_file_path,
        changed=False
    )

if __name__ == '__main__':
    main()

