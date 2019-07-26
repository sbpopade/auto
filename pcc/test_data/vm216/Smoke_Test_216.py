##########################################
#
# Test Data File for the PCC Smoke Test
#
##########################################


 # PCC Login Data
# Login into PCC mentioned into Server URL
# E.g. Here it will be Login into PCC-216
server_url = "https://172.17.2.216:9999"
user_name = "admin"
user_pwd = "admin"

 #Node data
total_invader = 1
total_server = 2

 # Test Data for Invader as Node
# Update This data as per supported Invader over PCC server
invader_node_name = "i58"
invader_node_host = "172.17.2.58"


 # Test Data to Add Server as Node
# Update This data as per supported Server over PCC
server1_node_name = "sv8"
server1_node_host = "172.17.2.101"
server1_bmc_host = "172.17.3.101"
server1_bmc_user = "ADMIN"
server1_bmc_pwd = "ADMIN"
server1_console = "ttyS1"
server1_managed_by_pcc = "true"
server1_ssh_keys = "pcc"

server2_node_name = "sv8"
server2_node_host = "172.17.2.102"
server2_bmc_host = "172.17.3.102"
server2_bmc_user = "ADMIN"
server2_bmc_pwd = "ADMIN"
server2_console = "ttyS1"
server2_managed_by_pcc = "true"
server2_ssh_keys = "pcc"


total_group = 2
# Test Data for Group Creation
create1_group_name = "automation_group1"
create1_group_desc = "automation_group1"
create2_group_name = "automation_group2"
create2_group_desc = "automation_group2"


# Assign Group To Node
# Please make sure group name is present in group list
# before select group name
# By Default keep it as "automation_group" as we are creating this
# group just before group assignment test
assign_group_name = "automation_group1"


total_role = 2
# Node Role Creation Data
# It will assign ROOT as role Tenant
create1_role_name = "automation_role1"
create1_role_desc = "automation_role1"
create2_role_name = "automation_role2"
create2_role_desc = "automation_role2"

# Assign Roles To Node
# Please make sure role name is present in role list
# before select group role
# By Default it will assign "LLDP" roles to node
assign_role_name = "LLDP"


total_site = 2
# Site Creation Data
create1_site_name = "automation_site1"
create1_site_desc = "automation_site1"
create2_site_name = "automation_site2"
create2_site_desc = "automation_site2"

# Assign Sites To Node
# Please make sure Site name is present in Site list
# before select Site name
# By Default it will assign "automation_site" site to node
assign_site_name = "automation_site1"


total_tenant = 2
# Tenant Creation Data
create1_tenant_name = "automation_tenant1"
create1_tenant_desc = "automation_tenant1"
create2_tenant_name = "automation_tenant2"
create2_tenant_desc = "automation_tenant2"

 # Assign Tenant To Node
# Please make sure Tenant name is present in Tenant list
# before select Tenant name
# By Default it will assign "automation_tenant" tenant to node
assign_tenant_name = "automation_tenant1"
