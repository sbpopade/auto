#!/usr/bin/python
""" Add/Remove Advertised Route """

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

from ansible.module_utils.basic import AnsibleModule

from getmac import get_mac_address

DOCUMENTATION = """
---
module: add_remove_route
author: Platina Systems
short_description: Module to add/remove advertised route.
description:
    Module to add/remove advertised route for BGP.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    remove:
      description:
        - Flag to indicate if we want to add a route or remove it.
      required: False
      type: bool
      default: False
"""

EXAMPLES = """
- name: Add advertised route
  initiate_traffic:
    switch_name: "{{ inventory_hostname }}"
    remove: False

- name: Remove advertised route
  initiate_traffic:
    switch_name: "{{ inventory_hostname }}"
    remove: True
"""

RETURN = """
hash_dict:
  description: Dictionary containing key value pairs to store in hash.
  returned: always
  type: dict
"""


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
    key = '{}  {}'.format(exec_time, cmd)
    HASH_DICT[key] = out

    return out

def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            mac_address=dict(required=False,type='str'),
            target_switch=dict(required=False, type='str'),
        )
    )

    target_switch = module.params['target_switch']
    mac_address = module.params['mac_address']
    switch_name = module.params['switch_name']
    print "################# Pointer ###############"
    print mac_address
    if switch_name==target_switch:
        cmd = "python -m getmac -i 'xeth4'"
        mac_address = run_cli(module, cmd) 
        print mac_address
    else:
        ip = '10.3.1.{}'.format(target_switch[-2::])
	print "################# Pointer1 ###############"
	cmd ="sudo ~/tmp/bcm.py 'ps ce3'"
	abc=run_cli(module, cmd)
	print abc
	cmd ="sudo ~/tmp/bcm.py 'cint stoploop.cint'"
	stop_stream=run_cli(module, cmd)
	print stop_stream

    msg = abc

    # Exit the module and return the required JSON.
    module.exit_json(
        msg=msg
    )

if __name__ == '__main__':
    main()


