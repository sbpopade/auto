############################################
#
# Test Data File for the PCC Entry Criteria
#
############################################


# PCC Login Data
# Login into PCC mentioned into Server URL
# E.g. Here it will be Login into PCC-216
server_url = "https://172.17.2.212:9999"
user_name = "admin"
user_pwd = "admin"

total_node = 2
total_server = 4

# Test Data for Invader as Node
# Update This data as per supported Invader over PCC server
invader1_node_name = "i42"
invader1_node_host = "172.17.2.42"

invader2_node_name = "i43"
invader2_node_host = "172.17.2.43"


 # Test Data to Add Server as Node
# Update This data as per supported Server over PCC
server1_node_name = "sv0"
server1_node_host = "172.17.2.110"
server1_bmc_host = "172.17.3.110"
server1_bmc_user = "ADMIN"
server1_bmc_pwd = "ADMIN"
server1_console = "ttyS1"
server1_managed_by_pcc = "true"
server1_ssh_keys = "pcc"

server2_node_name = "sv1"
server2_node_host = "172.17.2.111"
server2_bmc_host = "172.17.3.111"
server2_bmc_user = "ADMIN"
server2_bmc_pwd = "ADMIN"
server2_console = "ttyS1"
server2_managed_by_pcc = "true"
server2_ssh_keys = "pcc"

server3_node_name = "sv24"
server3_node_host = "172.17.2.124"
server3_bmc_host = "172.17.3.124"
server3_bmc_user = "ADMIN"
server3_bmc_pwd = "ADMIN"
server3_console = "ttyS1"
server3_managed_by_pcc = "true"
server3_ssh_keys = "pcc"

server4_node_name = "sv25"
server4_node_host = "172.17.2.125"
server4_bmc_host = "172.17.3.125"
server4_bmc_user = "ADMIN"
server4_bmc_pwd = "ADMIN"
server4_console = "ttyS1"
server4_managed_by_pcc = "true"
server4_ssh_keys = "pcc"

 # Credentials to access invader via ssh
invader_usr_name = "pcc"
invader_usr_pwd = "cals0ft"


 # OS Deployment data
image_name = "centos76"
en_US = "en_US"
PDT = "PDT"
mass_user = "auto_test"
ssh_key = "['pcc']"