*** Settings ***
Library  	      	OperatingSystem
Library  	      	Collections
Library  	      	String

Library    	      	${CURDIR}/../lib/Request.py
Variables         	${CURDIR}/../test_data/Url_Paths.py
Library           	${CURDIR}/../lib/Data_Parser.py
Resource          	${CURDIR}/../resource/Resource_Keywords.robot

Test Setup       	Verify User Login
Test Teardown    	Delete All Sessions


*** Test Cases ***
Add Invader as a Node
        [Tags]    Smoke_Test    Node Management
        [Documentation]    Verify User Should be able to add Invader as Node

        # Add Invader Node
        &{data}    Create Dictionary  	Name=${invader_node_name}  Host=${invader_node_host}
        Log    \nCreating Invader Node with parameters : \n${data}\n    console=yes
        ${resp}    Post Request    platina    ${add_node}    json=${data}   headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # wait for few seconds to add Invader into Node List
        Sleep    90s

        # Validate Added Node Present in Node List
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        # Hit get_node_list API for few times to refresh the node list
        # And verify Node availability from the latest fetched node data
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Sleep    3s
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Sleep    3s
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # Parse fetched node list and verify added Node availability from response data
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${invader_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${invader_node_name} is not present in node list
        Log    \n Invader ${invader_node_name} ID = ${node_id}   console=yes
        Set Suite Variable    ${invader_id}    ${node_id}

        # Verify Online Status of Added Invader
        ${status}    Validate Node Online Status    ${resp.json()}    ${invader_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${invader_node_name} added successfully but it is offline


Add Server as a Node
        [Tags]    Smoke_Test    Node Management
        [Documentation]    Verify User Should be able to add Server as Node

        # Add Server Node
        @{server_bmc_users}    Create List    ${server_bmc_user}
        @{server_ssh_keys}    Create List    ${server_ssh_keys}
        &{data}    Create Dictionary  	Name=${server_node_name}  Host=${server_node_host
        ...    console=${server_console}  bmc=${server_bmc_host}  bmcUser=${server_bmc_user}
        ...    bmcPassword=${server_bmc_pwd}  bmcUsers=@{server_bmc_users}
        ...    sshKeys=@{server_ssh_keys}  managed=${${server_managed_by_pcc}}
        Log    \nCreating Server node with parameters : \n${data}\n    console=yes
        ${resp}    Post Request    platina    ${add_node}    json=${data}   headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # wait for few seconds to add Server into Node List
        Sleep    90s

        # Validate Added Node Present in Node List
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        # Hit get_node_list API for few times to refresh the node list
        # And verify Node availability from the latest fetched node data
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Sleep    3s
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Sleep    3s
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # Parse fetched node list and verify added Node availability from response data
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${server_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${server_node_name} is not present in node list
        Log    \n Server ${server_node_name} ID = ${node_id}   console=yes
        Set Suite Variable    ${server_id}    ${node_id}

        # Verify Online Status of Added Server
        ${status}    Validate Node Online Status    ${resp.json()}    ${server_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${server_node_name} added successfully but it is offline


Create a Node Role
        [Tags]    Smoke_Test    Roles
        [Documentation]    Node Role Creation

        # Create a Node Role
        &{data}    Create Dictionary  name=${create_role_name}    description=${create_role_desc}    owner=${1}
        Log    \nCreating Node Role With Params: \n${data}\n    console=yes
        ${resp}  Post Request    platina   ${add_role}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # Wait for few seconds to reflect node roles over PCC
        Sleep    5s

        # Validate Added Role
        ${resp}  Get Request    platina   ${add_role}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${role_id}    Validate Roles    ${resp.json()}    ${create_role_name}
        Should Be Equal As Strings    ${status}    True    msg=Role ${create_role_name} is not present in Roles list
        Set Suite Variable    ${create_role_id}    ${role_id}
        Log    \n Roles ${create_role_name} ID = ${create_role_id}   console=yes


PCC Node Role Assignment
        [Tags]    Smoke_Test    Roles
        [Documentation]    Node Role Assignment

        # Validate Assign Role is present in Roles list
        ${resp}  Get Request    platina   ${add_role}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${role_id}    Validate Roles    ${resp.json()}    ${assign_role_name}
        Should Be Equal As Strings    ${status}    True    msg=Role ${assign_role_name} is not present in Roles list
        Set Suite Variable    ${assign_role_id}    ${role_id}
        Log    \n Roles ${assign_role_name} ID = ${assign_role_id}   console=yes

        # Assign Role to Node
        &{data}    Create Dictionary  Id=${invader_id}    roles=${assign_role_id}
        Log    \nAssigning a Roles with parameters: \n${data}\n    console=yes
        ${resp}  Put Request    platina    ${add_group_to_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few seconds to reflect assign roles over node
        Sleep	3 minutes

        # Validate Assigned Roles
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node_id}    Validate Node Roles    ${resp.json()}    ${invader_node_name}    ${assign_role_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${invader_node_name} is not updated with the Roles ${assign_role_name}


Create a Site
        [Tags]    Smoke_Test    Sites
        [Documentation]    Verify User Should Be able to create Site

        # Add Site
        &{data}    Create Dictionary  Name=${create_site_name}    Description=${create_site_desc}
        Log    \nCreating a Site with parameters: \n${data}\n    console=yes
        ${resp}  Post Request    platina    ${add_site}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few seconds to reflect site name over PCC
        Sleep    5s

        # Validate Added Site
        ${resp}  Get Request    platina   ${get_site}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${site_id}    Validate Sites    ${resp.json()}    ${create_site_name}
        Should Be Equal As Strings    ${status}    True    msg=Site ${create_site_name} is not present in Site list
        Set Suite Variable    ${create_site_id}    ${site_id}
        Log    \n Site ${create_site_name} ID = ${create_site_id}   console=yes


Assign a Site to Node
        [Tags]    Smoke_Test    Sites
        [Documentation]    Assign a particular site to a Node

        # Validate Assign Site is present in Site list
        ${resp}  Get Request    platina   ${get_site}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${site_id}    Validate Sites    ${resp.json()}    ${assign_site_name}
        Should Be Equal As Strings    ${status}    True    msg=Site ${assign_site_name} is not present in Site list
        Set Suite Variable    ${assign_site_id}    ${site_id}
        Log    \n Site ${assign_site_name} ID = ${assign_site_id}   console=yes

        # Update Site With Node
        &{data}    Create Dictionary  Id=${invader_id}    Site_Id=${assign_site_id}
        Log    \nAssigning Site with params:\n${data}\n    console=yes
        ${resp}  Put Request    platina    ${add_site_to_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        Sleep    60s

        # Validated Updated Site
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node1_id}    Validate Node Site    ${resp.json()}    ${invader_node_name}    ${assign_site_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${invader_node_name} is not updated with the site ${assign_site_name}


Create a Tenant
        [Tags]    Smoke_Test    Tenant
        [Documentation]    Adding a new Tenant

        # Create Tenant
        &{data}    Create Dictionary    name=${create_tenant_name}   description=${create_tenant_desc}    parent=${1}
        Log    \nCreating Tenant with Params:\n${data}\n    console=yes
        ${resp}    Post Request    platina    ${add_tenant}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few seconds to reflect tenant over PCC
        Sleep    10s

        # Verify added tenant present in tenant list
        ${resp}  Get Request    platina    ${tenant_list}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200
        ${status}    ${tenant_id}    Get Tenant Id    ${resp.json()}    ${create_tenant_name}
        Should Be Equal As Strings    ${status}    True    msg=Tenant ${create_tenant_name} is not present in tenant list
        Set Suite Variable    ${create_tenant_id}    ${tenant_id}
        Log    \n Tenant ${create_tenant_name} ID = ${create_tenant_id}    console=yes


Assign a Tenant to Node
        [Tags]   Smoke_Test    Tenant
        [Documentation]    Assign Tenant to a Node

        # Verify assign tenant present in tenant list
        ${resp}  Get Request    platina   ${tenant_list}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${assign_tenant_id}    Get Tenant Id    ${resp.json()}    ${assign_tenant_name}
        Log    \n Tenant ${assign_tenant_name} ID = ${assign_tenant_id}    console=yes

        # Update Node With Tenants
        @{node_id_list}    Create List    ${invader_id}
        &{data}    Create Dictionary    tenant=${assign_tenant_id}    ids=@{node_id_list}
        ${resp}    Post Request    platina    ${node_tenant_assignment}    json=${data}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        # Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few seconds to reflect assign Tenant over Node
        Sleep    10s

        # Validated Updated Node with Tenant
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${invader_id}    Validate Node Tenant    ${resp.json()}    ${invader_node_name}    ${assign_tenant_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${invader_node_name} is not updated with the Tenant ${assign_tenant_name}


Create a Node Group
        [Tags]    Smoke_Test    Groups
        [Documentation]    Verify User Should be able to Create Node Group

        # Add Group
        &{data}    Create Dictionary  Name=${create_group_name}    Description=${create_group_desc}
        Log    \nCreating Group with parameters: \n${data}\n    console=yes
        ${resp}  Post Request    platina   ${add_group}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        Sleep    5s

        # Validate added group present in Group List
        ${resp}  Get Request    platina   ${get_group}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # Parse fetched group list and verify added Group availability from response data
        ${status}    ${id}    Validate Group    ${resp.json()}    ${create_group_name}
        Should Be Equal As Strings    ${status}    True    msg=Group ${create_group_name} is not present in Groups list
        Set Suite Variable    ${create_group_id}    ${id}
        Log    \n Group ${create_group_name} ID = ${create_group_id}   console=yes


PCC Node Group Assignment
        [Tags]    Smoke_Test    Groups
        [Documentation]    Node to Group Assignment

        # Verify Group is present before assign it to node
        ${resp}  Get Request    platina   ${get_group}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # Parse fetched group list and verify assign Group availability from response data
        ${status}    ${id}    Validate Group    ${resp.json()}    ${assign_group_name}
        Should Be Equal As Strings    ${status}    True    msg=Group ${assign_group_name} is not present in Groups list
        Set Suite Variable    ${assign_group_id}    ${id}
        Log    \n Group ${assign_group_name} ID = ${assign_group_id}    console=yes

        # Assign A Group to Node
        &{data}    Create Dictionary  Id=${invader_id}    ClusterId=${assign_group_id}
        Log    \nAssigning a Group with parameters: \n${data}\n    console=yes
        ${resp}  Put Request    platina    ${add_group_to_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few second to update Node with Assigned Group
        Sleep    60s

        # Validated Assigned Group
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node_id}    Validate Node Group    ${resp.json()}    ${invader_node_name}    ${assign_group_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${invader_node_name} is not updated with the Group ${assign_group_name}