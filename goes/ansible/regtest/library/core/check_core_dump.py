#!/usr/bin/python
""" Check for Core Dump """

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
import calendar

from collections import OrderedDict

from ansible.module_utils.basic import AnsibleModule

DOCUMENTATION = """
---
module: check_core_dump
author: Platina Systems
short_description: Module to check core dump in system logs.
description:
    Module to check core dump in system logs.
options:
    switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    start_time: 
      description:
        - The start time recorded in the starting of playbook execution.
      required: False
      type: str
    log_file_names:
      description:
        - Comma separated names of log files in which core dumps are to be checked(daemon.log, kern.log, syslog, dmesg).
      required: False
      type: str
    error_msg:
      description:
        - The error msg which is to be checked for in the log file.
      required: False
      type: str
"""

EXAMPLES = """
- name: Check for core dump in syslog
  test_goes_status:
    switch_name: "{{ inventory_hostname }}"
    start_time: "{{ hostvars['server_emulator']['start_time'] }}"
    log_file_names: "/var/log/syslog"
    error_msg: "panic"
  register: module_out
"""

RETURN = """
msg:
  description: String describing the core dumps in file, if found
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

    if out:
        HASH_DICT[key] = out[:512] if len(out.encode('utf-8')) > 512 else out
    else:
        HASH_DICT[key] = out

    return out


def verify_log_dumps(module):
    """
    Method to verify dumps in log files on invader.
    :param module: The Ansible module to fetch input parameters.
    """
    global RESULT_STATUS
    failure_summary = ''

    switch_name = module.params['switch_name']
    start_time = module.params['start_time']
    log_file_names = module.params['log_file_names'].split(',')
    error_msg = module.params['error_msg']

    # Getting the time stamp in log file's time stamp format
    month = start_time[4:6]
    month_abb = calendar.month_abbr[int(month)]
    date = start_time[6:8]
    if date[0] == '0':
        date = ' {}'.format(date[1])
    time = start_time[8:-2:]
    time_stamp = month_abb + ' ' + date + ' ' + time

    # Check for errors in log files
    for file_name in log_file_names:
        try:
            with open(file_name, 'r') as f:
                content = f.read()
            log_file = content.split('\n')

            core_dump = []

            for line in log_file:
                if time_stamp in line:
                    for i in range(log_file.index(line), len(log_file)):
                        if error_msg in log_file[i] and 'check_core_dump' not in log_file[i]:
                            core_dump.append(log_file[i])
                            RESULT_STATUS = False
                    break

            errors = '\n'.join(core_dump)
            if not RESULT_STATUS:
                failure_summary += 'On switch {} '.format(switch_name)
                failure_summary += 'runtime error found in {} file.\n'.format(file_name)
                failure_summary += 'Errors:\n{}\n'.format(errors)

        except IOError:
            failure_summary += "Unable to open {} file\n".format(file_name)

    return failure_summary


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            switch_name=dict(required=False, type='str'),
            start_time=dict(required=False, type='str'),
            log_file_names=dict(required=False, type='str'),
            error_msg=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str')
        )
    )

    global RESULT_STATUS, HASH_DICT
    failure_summary = verify_log_dumps(module)

    # Calculate the entire test result
    HASH_DICT['result.detail'] = failure_summary
    HASH_DICT['result.status'] = 'Passed' if RESULT_STATUS else 'Failed'
    found = False if RESULT_STATUS else True

    # Exit the module and return the required JSON.
    module.exit_json(
        hash_dict=HASH_DICT,
        found=found,
        summary=failure_summary
    )


if __name__ == '__main__':
    main()
