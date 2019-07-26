#!/usr/bin/python
""" Docker Up Down """

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

DOCUMENTATION = """
---
module: docker_updown
author: Platina Systems
short_description: Module to bring up and bring down docker containers.
description:
    Module to bring up and bring down docker containers.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    leaf_switch:
      description:
        - Name of leaf switch on which pcktgen interfaces needs to be turn down.
      required: False
      type: list
      default: []
    config_file:
      description:
        - Config details of docker container.
      required: False
      type: str
    state:
      description:
        - String describing if docker container has to be brought up or down.
      required: False
      type: str
      choices: ['up', 'down']
"""

EXAMPLES = """
- name: Bring up docker container
  docker_updown:
    config_file: "{{ lookup('file', '../../group_vars/{{ inventory_hostname }}/{{ item }}') }}"
    state: 'up'
"""

RETURN = """
msg:
  description: String describing docker container state.
  returned: always
  type: str
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
        return out
    elif err:
        return err
    else:
        return None


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            config_file=dict(required=False, type='str'),
            leaf_list=dict(required=False, type='str'),
            state=dict(required=False, type='str', choices=['up', 'down']),
        )
    )

    d_move = '~/./docker_move.sh'
    switch_name = module.params['switch_name']
    leaf_list = module.params['leaf_list']
    container_name = ''
    eth_list = []
    vlan_id = []
    config_file = module.params['config_file'].splitlines()

    for line in config_file:
        if 'container_name' in line:
            container_name = line.split()[1]
        elif 'interface' in line:
            eth = line.split()[1]
            eth_list.append(eth)
        elif 'vlan_id' in line:
            vlan = line.split()[1]
            vlan_id.append(vlan)

    if module.params['state'] == 'up':
        is_leaf = True if switch_name in leaf_list else False

        # Bring down interfaces that are connected to packet generator
        if is_leaf:
            for eth in [x for x in range(1, 33) if x % 2 == 0]:
                run_cli(module, 'ifconfig xeth{} down'.format(eth))

        # Bring up given interfaces in the docker container
        for i in range(len(eth_list)):
            cmd = 'ip link add link xeth{} name xeth{}.{} type vlan id {}'.format(eth_list[i], eth_list[i],
                                                                                      vlan_id[i], vlan_id[i])
            run_cli(module, cmd)

            cmd = 'ip link set up xeth{}.{}'.format(eth_list[i], vlan_id[i])
            run_cli(module, cmd)

            cmd = '{} up {} xeth{}.{} 192.168.{}.{}/24'.format(d_move, container_name, eth_list[i], vlan_id[i],
                                                                 vlan_id[i], switch_name[-2::])
            run_cli(module, cmd)
    else:
        # Bring down all interfaces in the docker container
        for i in range(len(eth_list)):
            cmd = '{} down {} xeth{}.{}'.format(d_move, container_name, eth_list[i], vlan_id[i])
            run_cli(module, cmd)

            cmd = 'ip link set down xeth{}.{}'.format(eth_list[i], vlan_id[i])
            run_cli(module, cmd)

            cmd = 'ip link del xeth{}.{}'.format(eth_list[i], vlan_id[i])
            run_cli(module, cmd)

    # Exit the module and return the required JSON.
    module.exit_json(
        msg='Module executed successfully'
    )


if __name__ == '__main__':
    main()

