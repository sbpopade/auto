#!/usr/bin/python
""" Test BMC Processor's Redis DB """

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
# along with Ansible.  If not, see <http://www.gnu.org/licenses/>.
#

import shlex

from collections import OrderedDict

from ansible.module_utils.basic import AnsibleModule


DOCUMENTATION = """
---
module: test_bmc_redis
author: Platina Systems
short_description: Module to test redis db on BMC processor.
description:
    Module to perform different tests on redis db running on BMC processor.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    bmc_redis_ip:
      description:
        - IP to access BMC processor's redis db.
      required: False
      type: str
    hash_name:
      description:
        - Name of the hash in which to store the result in redis.
      required: False
      type: str
    platina_redis_channel:
      description:
        - Name of the platina redis channel.
      required: False
      type: str
"""

EXAMPLES = """
- name: Test BMC Redis db
  test_bmc_redis:
    switch_name: "{{ inventory_hostname }}"
    bmc_redis_ip: "{{ ansible_ssh_host }}"
    platina_redis_channel: "platina-mk1"
"""

RETURN = """
hash_dict:
  description: Dictionary containing key value pairs to store in hash.
  returned: always
  type: dict
"""

RESULT_STATUS = True
HASH_DICT = OrderedDict()


def get_cli(module):
    """
    Method to get initial cli string.
    :param module: The Ansible module to fetch input parameters.
    :return: Initial cli string.
    """
    return '/home/platina/redis/redis-cli --raw -h {} '.format(module.params['bmc_redis_ip'])


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


def execute_and_verify(module, operation, param):
    """
    Execute hget command for the given parameter and verify the same.
    :param module: The Ansible module to fetch input parameters.
    :param operation: Name of the operation to perform: hget/hset.
    :param param: Name of the parameter.
    """
    global HASH_DICT, RESULT_STATUS
    failure_summary = ''
    switch_name = module.params['switch_name']
    
    cmd = '{} {} {} '.format(operation, 
                             module.params['platina_redis_channel'], param)

    cli = get_cli(module) + cmd
    out = run_cli(module, cli)

    # Store command prefixed with exec time as key and
    # command output as value in the hash dictionary
    exec_time = run_cli(module, 'date +%Y%m%d%T')
    key = '{0} {1} {2}'.format(switch_name, exec_time, cmd)
    HASH_DICT[key] = out

    # For errors, update the result status to False
    if out is None:
        RESULT_STATUS = False
        failure_summary += 'On switch {} '.format(switch_name)
        failure_summary += 'output of command {} is None\n'.format(cli)
    elif 'error' in out.lower():
        RESULT_STATUS = False
        failure_summary += 'On switch {} '.format(switch_name)
        failure_summary += 'output of command {} has errors\n'.format(cli)

    return failure_summary


def test_hget_operations(module):
    """
    Method to test basic hget operations on bmc redis db.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS
    failure_summary = ''

    parameter = 'temp'
    failure_summary += execute_and_verify(module, 'hget', parameter)

    parameter = 'status'
    failure_summary += execute_and_verify(module, 'hget', parameter)

    parameter = 'fan_tray'
    failure_summary += execute_and_verify(module, 'hget', parameter)

    parameter = 'psu'
    failure_summary += execute_and_verify(module, 'hget', parameter)

    parameter = 'vmon'
    failure_summary += execute_and_verify(module, 'hget', parameter)

    HASH_DICT['result.detail'] = failure_summary


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            bmc_redis_ip=dict(required=False, type='str'),
            platina_redis_channel=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
        )
    )

    global RESULT_STATUS, HASH_DICT

    # Perform and verify all required tests
    test_hget_operations(module)

    # Calculate the entire test result
    HASH_DICT['result.status'] = 'Passed' if RESULT_STATUS else 'Failed'

    # Exit the module and return the required JSON.
    module.exit_json(
        hash_dict=HASH_DICT
    )


if __name__ == '__main__':
    main()

