#!/usr/bin/python

""" Verify Vnetd """

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
module: test_ospf_multipath_routing
author: Platina Systems
short_description: Module to test and verify multipath routing using ospf.
description:
    Module to test and verify multipath routing using ospf and log the same.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    config_file:
      description:
        - ospf config which have been added.
      required: False
      type: str
    package_name:
      description:
        - Name of the package installed (e.g. gobgpd).
      required: False
      type: str
    leaf_list:
      description:
        - List of leaf invaders.
      required: False
      type: list
    spine_list:
      description:
        - List of spine invaders.
      required: False
      type: list
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
- name: Verify quagga bgp peering ebgp
  test_ospf_multipath_routing:
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

def vnetd_panic(module):

	xeth = module.params['xeth']

	global RESULT_STATUS, HASH_DICT
	failure_summary = ''
	execute_commands(module, "ip add sh xeth{}".format(xeth))
	
	out = execute_commands(module, "ping -c 5 10.0.{}.30".format(xeth))
	if "100% packet loss" in out:
		RESULT_STATUS = False
		failure_summary += "100% packet loss occured during ping from {} ".format(module.params['switch_name'])
		failure_summary += "for command {}\n".format("ping 10.0.{}.30".format(xeth))

	out = execute_commands(module, "sudo ip link del xeth{}".format(xeth))
	if "Operation not supported" not in out:
		RESULT_STATUS = False
		failure_summary += "Operation not supported is not present in the "
		failure_summary += "output of coommand {}\n".format("sudo ip link del xeth{}".format(xeth))

	# Check goes status
	execute_commands(module, "goes status")

	cmd = "ip link show xeth{}".format(xeth)
        out = execute_commands(module, cmd)
	if "xeth{}@".format(xeth) not in out:
		RESULT_STATUS = False
                failure_summary += "Operation not supported is not present in the "
                failure_summary += "output of coommand {}\n".format("sudo ip link del xeth{}".format(xeth))

	HASH_DICT['result.detail'] = failure_summary


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            xeth=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS

    vnetd_panic(module)

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

