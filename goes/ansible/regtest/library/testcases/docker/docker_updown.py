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
    config_file:
      description:
        - Config details of docker container.
      required: False
      type: str
    is_subports:
      description:
        - Flag to indicate if sub ports are used during port provisioning.
      required: False
      type: bool
      default: False
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
    switch_name: "{{ inventory_hostname }}"
    spine_list: "{{ groups['spine'] }}"
    leaf_list: "{{ groups['leaf'] }}"
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
            spine_list=dict(required=False, type='list', default=[]),
            leaf_list=dict(required=False, type='list', default=[]),
            config_file=dict(required=False, type='str'),
            is_subports=dict(required=False, type='bool', default=False),
            state=dict(required=False, type='str', choices=['up', 'down'])
        )
    )

    d_move = '~/./docker_move.sh'
    container_name = None
    eth_list = []
    subport = 1
    switch_name = module.params['switch_name']
    spine_list = module.params['spine_list']
    leaf_list = module.params['leaf_list']
    config_file = module.params['config_file'].splitlines()
    is_subports = module.params['is_subports']

    first_octet = ''
    is_leaf = True if switch_name in leaf_list else False

    for line in config_file:
        if 'container_name' in line:
            container_name = line.split()[1]
        elif 'interface' in line:
            eth = line.split()[1]
            eth_list.append(eth)
        elif 'subport' in line:
            subport = line.split()[1]

    container_id = container_name[1::]
    dummy_id = int(container_id)

    if module.params['state'] == 'up':
        # Add dummy interface and bring it up
        cmd = 'ip link add dummy{} type dummy 2> /dev/null'.format(dummy_id)
        run_cli(module, cmd)

        # Bring up dummy interface
        cmd = '{} up {} dummy{} 192.168.{}.1/32'.format(d_move, container_name,
                                                        dummy_id, container_id)
        run_cli(module, cmd)

        # Bring up given interfaces in the docker container
        for eth in eth_list:
            eth_is_even = True if int(eth) % 2 == 0 else False
            if not eth_is_even:
                if is_leaf:
                    if leaf_list.index(switch_name) == 0 and eth_list.index(eth) == 0:
                        first_octet = '10'

                    elif leaf_list.index(switch_name) == 0 and eth_list.index(eth) == 2:
                        first_octet = '13'

                    elif leaf_list.index(switch_name) == 1 and eth_list.index(eth) == 0:
                        first_octet = '12'

                    elif leaf_list.index(switch_name) == 1 and eth_list.index(eth) == 2:
                        first_octet = '11'

                else:
                    if spine_list.index(switch_name) == 0 and eth_list.index(eth) == 0:
                        first_octet = '10'

                    elif spine_list.index(switch_name) == 0 and eth_list.index(eth) == 2:
                        first_octet = '11'

                    elif spine_list.index(switch_name) == 1 and eth_list.index(eth) == 0:
                        first_octet = '12'

                    elif spine_list.index(switch_name) == 1 and eth_list.index(eth) == 2:
                        first_octet = '13'

            else:
                first_octet = '15'

            if not is_subports:
                cmd = '{} up {} xeth{} {}.{}.{}.{}/24'.format(
                    d_move, container_name, eth, first_octet, eth, subport, switch_name[-2::])
                run_cli(module, cmd)
            else:
                cmd = '{} up {} xeth{}-{} {}.{}.{}.{}/24'.format(
                    d_move, container_name, eth, subport, first_octet, eth, subport, switch_name[-2::])
                run_cli(module, cmd)
    else:
        # Bring down all interfaces in the docker container
        for eth in eth_list:
            cmd = '{} down {} xeth{}'.format(d_move, container_name,
                                                eth)
            run_cli(module, cmd)

        # Bring down dummy interface
        cmd = '{} down {} dummy{}'.format(d_move, container_name, dummy_id)
        run_cli(module, cmd)

    # Exit the module and return the required JSON.
    module.exit_json(
        msg='Module executed successfully'
    )


if __name__ == '__main__':
    main()
