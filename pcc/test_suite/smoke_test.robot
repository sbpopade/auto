
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
        [Tags]    Invader
        [Documentation]    Verify User Should be able to add Invader as Node

        :For  ${index}  IN RANGE    0  ${total_invader}
        \    ${Index}    Evaluate    ${index}+1
        \    Run Keyword And Continue On Failure    Add Invader    ${Index}


Add Server as a Node
        [Tags]    Server
        [Documentation]    Verify User Should be able to add Invader as Node

        :For  ${index}  IN RANGE    0  ${total_server}
        \    ${Index}    Evaluate    ${index}+1
        \    Run Keyword And Continue On Failure    Add Server    ${Index}


Create Site
        [Tags]    Site
        [Documentation]    Verify User Should be able to create site 
        
        :For  ${index}  IN RANGE    0  ${total_site}
        \    ${Index}    Evaluate    ${index}+1
        \    Run Keyword And Continue On Failure    Add Site    ${Index}


Create Group
        [Tags]    Group
        [Documentation]    Verify User Should be able to create group

        :For  ${index}  IN RANGE    0  ${total_group}
        \    ${Index}    Evaluate    ${index}+1
        \    Run Keyword And Continue On Failure    Add Group    ${Index}


Create Role
        [Tags]    Role
        [Documentation]    Verify User Should be able to create role

        :For  ${index}  IN RANGE    0  ${total_role}
        \    ${Index}    Evaluate    ${index}+1
        \    Run Keyword And Continue On Failure    Add Role    ${Index}


Create Tenant
        [Tags]    Tenant
        [Documentation]    Verify User Should be able to create tenant

        :For  ${index}  IN RANGE    0  ${total_tenant}
        \    ${Index}    Evaluate    ${index}+1
        \    Run Keyword And Continue On Failure    Add Tenant    ${Index}


Assign LLDP role to Invaders
        [Tags]    role_assign
        [Documentation]    Verify User Should be able to assign role to node

        :For  ${index}  IN RANGE    0  ${total_}
        \    ${Index}    Evaluate    ${index}+1
        \    Run Keyword And Continue On Failure    Add Tenant    ${Index}



*** keywords ***
Add Invader
	[Arguments]    ${Index}
 
        # Add Invader Node
	${name}     Set Variable  ${Invader${index}_node_name}
	${host}    Set Variable  ${Invader${index}_node_host}
        &{data}     Create Dictionary  	Name=${name}  Host=${host}
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
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${name} is not present in node list
        Log    \n Invader ${name} ID = ${node_id}   console=yes
        Set Suite Variable    ${invader${index}_id}    ${node_id}

        # Verify Online Status of Added Invader
        ${status}    Validate Node Online Status    ${resp.json()}    ${name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${name} added successfully but it is offline


Add Server
	[Arguments]    ${Index}

        # Add Server Node

	${name}    Set Variable  ${server${index}_node_name}
	${host}   Set Variable   ${server${index}_node_host}
	${bmc_host}   Set Variable  ${server${index}_bmc_host}
	${bmc_user}   Set Variable  ${server${index}_bmc_user}
	${bmc_pwd}   Set Variable  ${server${index}_bmc_pwd}
	${console}   Set Variable  ${server${index}_console}
	${manage_pcc}    Set Variable  ${server${index}_managed_by_pcc}
	${ssh_key}    Set Variable  ${server${index}_ssh_keys}


        @{server_bmc_users}    Create List    ${bmc_user}
        @{server_ssh_keys}    Create List    ${ssh_key}
        &{data}    Create Dictionary  	Name=${name}  Host=${host}
        ...    console=${console}  bmc=${bmc_host}  bmcUser=${bmc_user}
        ...    bmcPassword=${bmc_pwd}  bmcUsers=@{server_bmc_users}
        ...    sshKeys=@{server_ssh_keys}  managed=${${server${Index}_managed_by_pcc}}
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
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${name} is not present in node list
        Log    \n Server ID = ${node_id}   console=yes
        Set Suite Variable    ${server${Index}_id}    ${node_id}

        # Verify Online Status of Added Server
        ${status}    Validate Node Online Status    ${resp.json()}    ${name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${name} added successfully but it is offline
	

Add Site
       	[Arguments]    ${Index}

       	# Add Site
        &{data}    Create Dictionary  Name=${create${Index}_site_name}    Description=${create${Index}_site_desc}
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
        ${status}    ${site_id}    Validate Sites    ${resp.json()}    ${create${Index}_site_name}
        Should Be Equal As Strings    ${status}    True    msg=Site ${create${Index}_site_name} is not present in Site list
        Set Suite Variable    ${create${Index}_site_id}    ${site_id}
        Log    \n Site ${create${Index}_site_name} ID = ${create${Index}_site_id}   console=yes


Add Group
	[Arguments]    ${Index}

	# Add Group
        &{data}    Create Dictionary  Name=${create${Index}_group_name}    Description=${create${Index}_group_desc}
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
        ${status}    ${id}    Validate Group    ${resp.json()}    ${create${Index}_group_name}
        Should Be Equal As Strings    ${status}    True    msg=Group ${create${Index}_group_name} is not present in Groups list
        Set Suite Variable    ${create${Index}_group_id}    ${id}
        Log    \n Group ${create${Index}_group_name} ID = ${create${Index}_group_id}   console=yes

        
Add Role
        [Arguments]    ${Index}

        # Create a Node Role
        &{data}    Create Dictionary  name=${create${Index}_role_name}    description=${create${Index}_role_desc}    owner=${1}
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
        ${status}    ${role_id}    Validate Roles    ${resp.json()}    ${create${Index}_role_name}
        Should Be Equal As Strings    ${status}    True    msg=Role ${create${Index}_role_name} is not present in Roles list
        Set Suite Variable    ${create${Index}_role_id}    ${role_id}
        Log    \n Roles ${create${Index}_role_name} ID = ${create${Index}_role_id}   console=yes


Add Tenant
        [Arguments]    ${Index}

        # Create Tenant
        &{data}    Create Dictionary    name=${create${Index}_tenant_name}   description=${create${Index}_tenant_desc}    parent=${1}
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
        ${status}    ${tenant_id}    Get Tenant Id    ${resp.json()}    ${create${Index}_tenant_name}
        Should Be Equal As Strings    ${status}    True    msg=Tenant ${create${Index}_tenant_name} is not present in tenant list
        Set Suite Variable    ${create${Index}_tenant_id}    ${tenant_id}
        Log    \n Tenant ${create${Index}_tenant_name} ID = ${create${Index}_tenant_id}    console=yes


