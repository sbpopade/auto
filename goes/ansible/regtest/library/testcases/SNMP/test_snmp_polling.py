import shlex
import time

from collections import OrderedDict

from ansible.module_utils.basic import AnsibleModule

DOCUMENTATION = """
---
module: test_snmp_polling
author: Platina Systems
short_description: Module to test and verify multipath routing using ospf.
description:
    Module to test and verify multipath routing using ospf and log the same.
options:
    spine_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    package_name:
      description:
        - Name of the package installed (e.g. gobgpd).
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
- name: Verify snmp polling
  test_snmp_polling:
    spine_name: "{{ inventory_hostname }}"
    package_name: "snmp"
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

def snmp_polling(module, restart):

   global RESULT_STATUS, HASH_DICT
   failure_summary = ''

   spine = module.params['spine_name']
   package = module.params['package_name']
   
   cmd1 = "snmpwalk -v 2c -c public 172.17.2.{} sysName".format(spine[-2::])
   cmd2 = "snmpwalk -v 2c -c public 172.17.2.{} ifIndex".format(spine[-2::])
   cmd3 = "snmpwalk -v 2c -c public 172.17.2.{} ifHCInOctets".format(spine[-2::])
   cmd4 = "snmpwalk -v 2c -c public 172.17.2.{} ifHCOutOctets".format(spine[-2::])

   cmd_list = [cmd1, cmd2, cmd3, cmd4]

   for cmd in cmd_list:
	if not execute_commands(module, cmd):
		RESULT_STATUS = False
		failure_summary += "Unable to verify the output of {}".format(cmd)
                failure_summary += " command as this command do not executed successfully {} {} restart.\n".format(restart, package)

   HASH_DICT['result.detail'] = failure_summary

def main():
    """ This section is for arguments parsing """
    module = AnsibleModule(
        argument_spec=dict(
            spine_name=dict(required=False, type='str'),
            package_name=dict(required=False, type='str'),
            switch_name=dict(required=False, type='str'),
            hash_name=dict(required=False, type='str'),
            log_dir_path=dict(required=False, type='str'),
        )
    )

    global HASH_DICT, RESULT_STATUS

    snmp_polling(module, 'before')
    execute_commands(module, 'service {} restart'.format(module.params['package_name']))
    execute_commands(module, 'service {} status'.format(module.params['package_name']))
    snmp_polling(module, 'after')

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
