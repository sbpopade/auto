#!/usr/bin/python
""" Add/Delete Static Routes """

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
#This script calculates the passed failed and skipped/disabled test cases in the regression suite.
#It takes three arguments:
#first- host file name to indentify testbed.
#second & third- list of leaf and spine invader on which the regression executed.
#It also gather some additional information of the host involved in the run.
#It exports all the above gathering into a file(var_file.txt) in key value pairs.

import shlex

from ansible.module_utils.basic import AnsibleModule

DOCUMENTATION = """
---
module: summarize_regression
author: Platina Systems
short_description: Module to summarize regression result.
description:
    Module to summarize regression result.
options:
    testbed_name:
      description:
        - Name of the testbed on which regresion executed.
      required: False
      type: str
"""

EXAMPLES = """
- name: Summarize the regerssion suite result
  summarize_regression:
    testbed_name: {{ testbed_name }}
"""

RETURN = """
hash_dict:
  description: Dictionary containing key value pairs.
  returned: always
  type: dict
"""


def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            testbed_name=dict(required=False, type='str'),
            playbook_dir=dict(required=False, type='str'),
            goes_version=dict(required=False, type='str'),
            goes_build_id=dict(required=False, type='str'),
            goes_tags=dict(required=False, type='str'),
            goes_vnetd_id=dict(required=False, type='str'),
            kernel_version=dict(required=False, type='str')
        )
    )

    testbed_name = module.params['testbed_name']
    playbook_dir = module.params['playbook_dir']
    g_version = module.params['goes_version'].split('\n')
    g_build = module.params['goes_build_id'].split('\n')
    g_tags = module.params['goes_tags'].split('\n')
    v_build = module.params['goes_vnetd_id'].split('\n')
    k_version = module.params['kernel_version'].split('\n')
    goes_version = "\\n".join(g_version)
    goes_build_id = "\\n".join(g_build)
    goes_tags = "\\n".join(g_tags)
    goes_vnetd_id = "\\n".join(v_build)
    kernel_version = "\\n".join(k_version)

    regression_summary_file = '/var/log/regression/regression_summary_file_{}'.format(testbed_name)
    all_testcase_file = '{}/files/all_testcase_file'.format(playbook_dir)

    message = ''
    passed_testcase_list, failed_testcase_list, skipped_testcase_list = '', '', ''
    skipped_count, passed_count, failed_count = 0, 0, 0
    try:
        with open(all_testcase_file, 'r') as f:
            content = f.read()
        all_testcase_list = content.splitlines()

    except IOError:
        message += "Unable to open {} file\n".format(all_testcase_file)

    try:
        with open(regression_summary_file, 'r') as f:
            content = f.read()
        regression_summary_report = content.splitlines()

    except IOError:
        message += "Unable to open {} file\n".format(regression_summary_file)

    if message:
        module.exit_json(
            message=message
        )

    for line1 in all_testcase_list:
        found = False
        for line2 in regression_summary_report:
            if line1 == line2[:-25]:
                found = True
        if not found:
            skipped_testcase_list += '<li>{}</li>'.format(line1)
            skipped_count += 1

    for line in regression_summary_report:
        if 'Passed' in line:
            passed_testcase_list += '<li><a href="http://172.17.2.28/regression_logs/logs.html">{}</a></li>'.format(line)
            passed_count += 1
        elif 'Failed' in line:
            failed_testcase_list += '<li><a href="http://172.17.2.28/regression_logs/logs.html">{}</a></li>'.format(line)
            failed_count += 1

    total_count = failed_count + passed_count
    failure_percent = float(failed_count*100)/total_count
    failure_percent = round(failure_percent, 2)

    subject = 'Regression Status- Failure {}%, {} out of {} test cases failed'.format(failure_percent, failed_count, total_count)

    # Exit the module and return the required JSON.
    module.exit_json(
        passed_count=passed_count,
        passed_list=passed_testcase_list,
        failed_count=failed_count,
        failed_list=failed_testcase_list,
        skipped_count=skipped_count,
        skipped_list=skipped_testcase_list,
        goes_version=goes_version,
        goes_build_id=goes_build_id,
        goes_tags=goes_tags,
        goes_vnetd_id=goes_vnetd_id,
        kernel_version=kernel_version,
        subject=subject
    )


if __name__ == '__main__':
    main()



