#!/usr/bin/python
""" Test link status """

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
module: test_link_status
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
- name: Verify link status
  test_link_status:
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

    if 'service' in cmd and 'restart' in cmd or module.params['dry_run_mode']:
        out = None
    else:
        out = run_cli(module, cmd)

    # Store command prefixed with exec time as key and
    # command output as value in the hash dictionary
    exec_time = run_cli(module, 'date +%Y%m%d%T')
    key = '{0} {1} {2}'.format(module.params['switch_name'], exec_time, cmd)
    HASH_DICT[key] = out

    return out


def verify_link_status(module):
	global RESULT_STATUS, HASH_DICT
	failure_summary = ''
	aname = module.params['switch_name']
	stage = module.params['stage']
        speed = module.params['speed']
        f_ports = module.params['f_ports']
	eth_list = [1,3,5,7,9,11,13,15,17,19,21,23,25,27,29,31]
        for ele in f_ports:
	    try:
        	eth_list.remove(ele)
	    except Exception as e:
		pass
	if module.params['sub']:
		for eth in eth_list:
		    for subp in module.params['sub']:
			if speed:
				out = execute_commands(module, "goes hget platina-mk1 vnet.xeth{}-{}.speed".format(eth, subp))
	                        out = out.splitlines()
        	                for line in out:
                	                if speed not in line:
                        	                RESULT_STATUS = False
                                	        failure_summary += "link speed is not {} {} link flapping on invader{} for {}.\n".format(speed, stage, aname, line)

                	out = execute_commands(module, "goes hget platina-mk1 vnet.xeth{}-{}.link".format(eth, subp))
			out = out.splitlines()
			for line in out:
	                	if "true" not in line:
        	                	RESULT_STATUS = False
                	        	failure_summary += "link status is not true {} link flapping on invader{} for {}.\n".format(stage, aname, line)

	else:			
		for eth in eth_list:
	        	out = execute_commands(module, "goes hget platina-mk1 vnet.xeth{}.link".format(eth))
			if "true" not in out:
				RESULT_STATUS = False
				failure_summary += "xeth{} link status is not true {} link flapping on invader{}.\n".format(eth, stage, aname)


	execute_commands(module, 'goes status')

	HASH_DICT['result.detail'] = failure_summary


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
	    delay=dict(required=False, type='int', default=10),
            retries=dict(required=False, type='int', default=6),
	    sub=dict(required=False, type='list'),
	    stage=dict(required=False, type='str', default=''),
            speed=dict(required=False, type='str'),
            dry_run_mode=dict(required=False, type='bool', default=False),
            hash_name=dict(required=False, type='str'),
            f_ports=dict(required=False, type='list', default=[]),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS
    if module.params['dry_run_mode']:
	module.params['switch_name']
        cmds_list = []

        execute_commands(module, 'goes status')
        for key, value in HASH_DICT.iteritems():
            cmds_list.append(key)

        # Exit the module and return the required JSON.
        module.exit_json(
            cmds=cmds_list
        )
    else:

        verify_link_status(module)

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


