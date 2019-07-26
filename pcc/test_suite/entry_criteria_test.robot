*** Settings ***
Library  	    OperatingSystem
Library  	    Collections
Library  	    String
Library         SSHLibrary

Library    	    ${CURDIR}/../lib/Request.py
Variables       ${CURDIR}/../test_data/Url_Paths.py
Library         ${CURDIR}/../lib/Data_Parser.py
Library         ${CURDIR}/../lib/Entry_Criteria_Api.py
Resource        ${CURDIR}/../resource/Resource_Keywords.robot

Test Setup      Verify User Login
Test Teardown   Delete All Sessions


*** Test Cases ***
Invader and Server Cleanup from UI
        [Tags]    Entry Criteria
        [Documentation]    Get and Delete available Invader and Server from the PCC

        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Get Nodes Status code = ${resp.status_code}    console=yes
        Log    \n Get Nodes Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}  ${node_id}  ${node_data}  Get Available Node Data  ${resp.json()}
        Set Suite Variable    ${node_host}    ${node_data}
        Set Suite Variable    ${node_avail_status}    ${status}
        Pass Execution If	${node_avail_status}==False    No Nodes are available over PCC
        Log    \nDeleting the Node's from UI......     console=yes
        :FOR    ${id}    IN    @{node_id}
        \    @{data}    Create List    ${id}
        \    Log    \nDeleting Node with ID = ${id}    console=yes
        \    ${resp}    Post Request    platina    ${delete_node}    headers=${headers}    json=${data}
        \    Log    \n Status code = ${resp.status_code}    console=yes
        \    Log    \n Response = ${resp.json()}    console=yes
        \    Should Be Equal As Strings    ${resp.status_code}    200
        \    Sleep    60s


Invader and Server Cleanup from Back-End
        [Tags]    Entry Criteria
        [Documentation]    Get and Delete available Invader and Server from back-end

        Pass Execution If       ${node_avail_status}==False    No Nodes are available over PCC
        ${node_type}    Get Node Type    ${node_host}
        Log    Node Type Data = ${node_type}    console=yes
        ${status}    Node Clean up from Back-End Command    ${node_type}
        Should Be Equal As Strings    ${status}    True    msg=Failed to clean up Data from Node Back End


Add Invader-1 as a Node and Verify Online Status
        [Tags]    Entry Criteria
        [Documentation]    Add Invader-1 as a Node and Verify Online Status

        # Add Invader Node
        &{data}    Create Dictionary  	Name=${invader1_node_name}  Host=${invader1_node_host}
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
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${invader1_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${invader1_node_name} is not present in node list
        Log    \n Invader ${invader1_node_name} ID = ${node_id}   console=yes
        Set Suite Variable    ${invader1_id}    ${node_id}

        # Verify Online Status of Added Invader
        ${status}    Validate Node Online Status    ${resp.json()}    ${invader1_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${invader1_node_name} added successfully but it is offline


Add Invader-2 as a Node and Verify Online Status
        [Tags]    Entry Criteria
        [Documentation]    Add Invader-2 as a Node and Verify Online Status

        # Add Invader Node
        &{data}    Create Dictionary  	Name=${invader2_node_name}  Host=${invader2_node_host}
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
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${invader2_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${invader2_node_name} is not present in node list
        Log    \n Invader ${invader2_node_name} ID = ${node_id}   console=yes
        Set Suite Variable    ${invader2_id}    ${node_id}

        # Verify Online Status of Added Invader
        ${status}    Validate Node Online Status    ${resp.json()}    ${invader2_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Invader ${invader2_node_name} added successfully but it is offline


Assign MaaS Roles to Invader - 1
        [Tags]    Entry Criteria
        [Documentation]    Assign MaaS Roles to Invader - 1

        # Get Id of MaaS role
        ${resp}  Get Request    platina   ${add_role}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200
        ${status}    ${role_id}    Get MaaS Role Id    ${resp.json()}
        Should Be Equal As Strings    ${status}    True    msg=MaaS Role Not Found in Roles
        Set Suite Variable    ${maas_role_id}    ${role_id}
        Log    \n MaaS Role ID = ${maas_role_id}    console=yes

        # Get Id of LLDP role
        ${resp}  Get Request    platina   ${add_role}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200
        ${status}    ${role_id}    Get LLDP Role Id    ${resp.json()}
        Should Be Equal As Strings    ${status}    True    msg=LLDP Role Not Found in Roles
        Set Suite Variable    ${lldp_role_id}    ${role_id}
        Log    \n LLDP Role ID = ${lldp_role_id}    console=yes

        # Assign MaaS role to node - 1
        @{roles_group}    create list    ${lldp_role_id}    ${maas_role_id}
        &{data}    Create Dictionary  Id=${invader1_id}    roles=${roles_group}
        ${resp}  Put Request    platina    ${add_group_to_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few Seconds
        Sleep    150s

        # SSH into invader and verify MaaS installation process started
        Run Keyword And Ignore Error	SSH into Invader and Verify mass installation started    ${invader1_node_host}

        # Wait for 12 minutes
        Sleep	12 minutes

        # Verify Maas Installation Complete status
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node_id}    Validate Node Roles    ${resp.json()}    ${invader1_node_name}    ${maas_role_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${invader1_node_name} is not updated with the MaaS Roles


Assign MaaS Roles to Invader - 2
        [Tags]    Entry Criteria
        [Documentation]    Assign MaaS Roles to Invader - 2

        # Assign MaaS role to node - 2
        @{roles_group}    create list    ${lldp_role_id}    ${maas_role_id}
        &{data}    Create Dictionary  Id=${invader2_id}    roles=${roles_group}
        ${resp}  Put Request    platina    ${add_group_to_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few Seconds
        Sleep    150s

        # SSH into invader and verify MaaS installation process started
        Run Keyword And Ignore Error	SSH into Invader and Verify mass installation started    ${invader2_node_host}

        # Wait for 12 minutes
        Sleep	12 minutes

        # Verify Maas Installation Complete status
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node_id}    Validate Node Roles    ${resp.json()}    ${invader2_node_name}    ${maas_role_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${invader2_node_name} is not updated with the MaaS Roles


PXE Boot to Server
        [Tags]    Entry Criteria
        [Documentation]    Server PXE Boot
        ${status}    Server PXE boot    ${server1_bmc_host}
        Should Be Equal As Strings    ${status}    True    msg=PXE boot Failed Over Server ${server1_node_host}
        # Wait till Server Get Boot
        Log    \nPXE boot Started......
        Sleep   20 minutes


Update Server information added after PXE boot
        [Tags]    Entry Criteria
        [Documentation]    Update Server information

        # Get Server ID
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node_id}    Get Server Id    ${resp.json()}    ${server1_bmc_host}
        Should Be Equal As Strings    ${status}    True    msg=Booted Server ${server1_bmc_host} is not present in node list
        Log    \n Server ${server1_node_name} ID = ${node_id}   console=yes
        Set Suite Variable    ${server1_id}    ${node_id}

        # Update Server Node with proper information
        @{server1_bmc_users}    Create List    ${server1_bmc_user}
        @{server1_ssh_keys}    Create List    ${server1_ssh_keys}
        &{data}    Create Dictionary    Id=${server1_id}  Name=${server1_node_name}  console=${server1_console}
        ...    managed=${${server1_managed_by_pcc}}  bmc=${server1_bmc_host}  bmcUser=${server1_bmc_user}
        ...    bmcPassword=${server1_bmc_pwd}  bmcUsers=@{server1_bmc_users}
        ...    sshKeys=@{server1_ssh_keys}  Host=${server1_node_host}
        Log    \n Updating Server with Data : \n${data}\n    console=yes
        ${resp}  Put Request    platina    ${update_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        # Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # wait for few seconds
        Sleep    90s

        # Validate Updated Server Present in Node List
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        # And verify Node availability from the latest fetched node data
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200

        # Parse fetched node list and verify added Node availability from response data
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${server1_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${server1_node_name} is not present in node list
        Log    \n Server ${server1_node_name} ID = ${node_id}   console=yes
        Set Suite Variable    ${server1_id}    ${node_id}

        # Verify Online Status of Added Server
        ${status}    Validate Node Online Status    ${resp.json()}    ${server1_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${server1_node_name} added successfully but it is offline


OS Deployment over Server machine
        [Tags]    Entry Criteria
        [Documentation]    OS Deployment
        # Start OS Deployment
        &{data}    Create Dictionary  nodes=[${${server1_id}}]  image=${image_name}  locale=${en_US}  timezone=${PDT}  adminUser=${mass_user}  sshKeys=${ssh_key}
        ${resp}  Post Request    platina   ${os_deployment}    json=${data}    headers=${headers}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
    	Should Be Equal As Strings  ${resp.status_code}  200

        # Wait for 15 minutes
        Sleep	15 minutes

        # Verify Provision Status over server
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    Validate Node Provision Status    ${resp.json()}    ${server_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Provision Status of server ${server_node_name} is not Finished

        # Verify CentOS installed in remote machine
        Verify CentOS installed in server machine


Assign LLDP role to Server
        [Tags]    Entry Criteria
        [Documentation]    Assign LLDP to Server

        # Assign Role to Node
        &{data}    Create Dictionary  Id=${server1_id}    roles=${lldp_role_id}
        Log    \nAssigning a Roles with parameters: \n${data}\n    console=yes
        ${resp}  Put Request    platina    ${add_group_to_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few seconds to reflect assign roles over node
        Sleep	5 minutes

        # Validate Assigned Roles
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node_id}    Validate Node Roles    ${resp.json()}    ${server_node_name}    ${lldp_role_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${server_node_name} is not updated with the Roles LLDP


Create Kubernetes Cluster
        [Tags]    Entry Criteria
        [Documentation]    Create Kubernetes Cluster

        # Create Kubernetes cluster
        &{data1}    Create Dictionary  id=${${invader1_id}}
        &{data2}    Create Dictionary  id=${${invader2_id}}
        &{data3}    Create Dictionary  id=${${server1_id}}
        @{json}    Create List    ${data1}  ${data2}  ${data3}

        &{data}    Create Dictionary  name=${cluster_name}  k8sVersion=${cluster_version}  cniPlugin=${cni_plugin}
        ...    nodes=@{json}
        Log    \nCreating Cluster with data: ${data}\n
        ${resp}  Post Request    platina   ${add_kubernetes_cluster}    json=${data}    headers=${headers}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n JSON RESP = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  200

        # Wait for few seconds
        Sleep  5 minutes
        Log To Console    \nK8s is installing..... 
        Sleep  5 minutes
        Log To Console    \nK8s is installing..... 
        Sleep  5 minutes
        Log To Console    \nK8s is installing..... 
        Sleep  5 minutes

        # Verify cluster created
        ${resp}  Get Request    platina   ${add_kubernetes_cluster}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${id}    Validate Cluster    ${resp.json()}    ${cluster_name}
        Should Be Equal As Strings    ${status}    True    msg=Created Cluster ${cluster_name} is not present in Cluster list
        Set Suite Variable    ${cluster_id}    ${id}
        ${status}    Validate Cluster Deploy Status    ${resp.json()}
        Should Be Equal As Strings    ${status}    True    msg=Cluster installation failed


Verify Created K8s Cluster installation from back end
        [Tags]    Entry Criteria
        [Documentation]    Verify Kubernetes Cluster

        ${status1}    Verify Kubernetes cluster installed    ${invader1_node_host}
        ${status2}    Verify Kubernetes cluster installed    ${invader2_node_host}
        ${status}    get k8s installation status    ${status1}    ${status2}
        Should Be Equal As Strings    ${status}    True    msg=K8s Cluster not running in back end


Add an app to k8s
        [Tags]    Entry Criteria
        [Documentation]    Add an App to K8S

        # Add an App to Kubernetes cluster
        &{data}    Create Dictionary  appName=${app_name}  appNamespace=${app_name}  gitUrl=${git_url}
        ..  gitRepoPath=${git_repo_path}  gitBranch=${git_branch}
        Log    \nAdding App with Data: ${data}\n
        ${resp}  Post Request    platina   ${add_kubernetes_cluster}${cluster_id}/app    json=${data}    headers=${headers}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  200

        Sleep    2 minutes

        # Verify App Installed Successfully..
        ${resp}  Get Request    platina   ${add_kubernetes_cluster}/${${cluster_id}}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    Verify App Present in Cluster    ${resp.json()}    ${app_name}
        Should Be Equal As Strings    ${status}    True    msg=Installed App ${app_name} is not present/installed in cluster


Add Server as a Node
        [Tags]    Entry Criteria
        [Documentation]    Add One More server as a Node

        # Add Server Node
        @{server_bmc_users}    Create List    ${server2_bmc_user}
        @{server_ssh_keys}    Create List    ${server2_ssh_keys}
        &{data}    Create Dictionary  	Name=${server2_node_name}  Host=${server2_node_host
        ...    console=${server2_console}  bmc=${server2_bmc_host}  bmcUser=${server2_bmc_user}
        ...    bmcPassword=${server2_bmc_pwd}  bmcUsers=@{server2_bmc_users}
        ...    sshKeys=@{server2_ssh_keys}  managed=${${server2_managed_by_pcc}}
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
        ${status}    ${node_id}    Validate Node    ${resp.json()}    ${server2_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${server2_node_name} is not present in node list
        Log    \n Server ${server_node_name} ID = ${node_id}   console=yes
        Set Suite Variable    ${server2_id}    ${node_id}

        # Verify Online Status of Added Server
        ${status}    Validate Node Online Status    ${resp.json()}    ${server2_node_name}
        Should Be Equal As Strings    ${status}    True    msg=Server ${server2_node_name} added successfully but it is offline


Assign LLDP role to Server
        [Tags]    Entry Criteria
        [Documentation]    Assign LLDP to Server

        # Assign Role to Node
        &{data}    Create Dictionary  Id=${server2_id}    roles=${lldp_role_id}
        Log    \nAssigning a Roles with parameters: \n${data}\n    console=yes
        ${resp}  Put Request    platina    ${add_group_to_node}    json=${data}     headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}    200

        # Wait for few seconds to reflect assign roles over node
        Sleep	5 minutes

        # Validate Assigned Roles
        &{data}    Create Dictionary  page=0  limit=50  sortBy=name  sortDir=asc  search=
        ${resp}  Get Request    platina   ${get_node_list}    params=${data}  headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    ${node_id}    Validate Node Roles    ${resp.json()}    ${server2_node_name}    ${lldp_role_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${server2_node_name} is not updated with the Roles LLDP


Add A Node To K8s
        [Tags]    Entry Criteria
        [Documentation]    Add an Node to K8S

        # Add A Node to Kubernetes cluster
        &{node_data}    Create Dictionary  id=${${server2_id}}
        &{data}    Create Dictionary  rolePolicy=auto  toAdd=[&{node_data}]  toRemove=[]
        Log    \nAdding Node with Data: ${data}\n
        ${resp}  Put Request    platina   ${add_kubernetes_cluster}${cluster_id}    json=${data}    headers=${headers}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  200

        Sleep    5 minutes

        # Verify Node Added Successfully..
        ${resp}  Get Request    platina   ${add_kubernetes_cluster}/${${cluster_id}}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    Verify Node Added in Cluster    ${resp.json()}    ${server2_id}
        Should Be Equal As Strings    ${status}    True    msg=Node ${server2_node_name} is not present in cluster


Delete a node from k8s
        [Tags]    Entry Criteria
        [Documentation]    Delete a Node From K8s

        # Delete Node From Kubernetes cluster
        &{data}    Create Dictionary  rolePolicy=auto  toAdd=[]  toRemove=[${${server1_id}}]
        Log    \nDeleting Node with Data: ${data}\n
        ${resp}  Put Request    platina   ${add_kubernetes_cluster}${cluster_id}    json=${data}    headers=${headers}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  200

        Sleep    5 minutes

        # Verify Node Deleted From Cluster
        ${resp}  Get Request    platina   ${add_kubernetes_cluster}/${${cluster_id}}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    Verify Node Added in Cluster    ${resp.json()}    ${server1_id}
        Should Be Equal As Strings    ${status}    False    msg=Node ${server1_node_name} is not deleted incluster


Update k8s and verify version updated
        [Tags]    Entry Criteria
        [Documentation]    Update K8s version

        Log    \nversion Update Step is Not automated as not able to perform it manually...\n


Install another app as sanity check
        [Tags]    Entry Criteria
        [Documentation]    Insatall Another app as sanity check

        # Install an App to K8s Cluster
        &{data}    Create Dictionary  appName=${app2_name}  appNamespace=${app2_name}  gitUrl=${git2_url}
        ..  gitRepoPath=${git2_repo_path}  gitBranch=${git2_branch}
        Log    \nAdding App with Data: ${data}\n
        ${resp}  Post Request    platina   ${add_kubernetes_cluster}${cluster_id}/app    json=${data}    headers=${headers}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  200

        Sleep    2 minutes

        # Verify App Installed Successfully..
        ${resp}  Get Request    platina   ${add_kubernetes_cluster}/${${cluster_id}}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    Verify App Present in Cluster    ${resp.json()}    ${app2_name}
        Should Be Equal As Strings    ${status}    True    msg=Installed App ${app2_name} is not present/installed in cluster


Delete K8s Cluster
        # Delete K8s Cluster
        Log    \nAdding App with Data: ${data}\n
        ${resp}  Delete Request	   platina    ${add_kubernetes_cluster}${cluster_id}    json={"forceRemove":false}    headers=${headers}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  200

        # Wait for few seconds
        Sleep    3 minutes

        # Verify Cluster Deleted
        ${resp}  Get Request    platina   ${add_kubernetes_cluster}    headers=${headers}
        Log    \n Status code = ${resp.status_code}    console=yes
        Log    \n Response = ${resp.json()}    console=yes
        Should Be Equal As Strings    ${resp.status_code}    200
        ${status}    Verify Cluster Deleted    ${resp.json()}    ${cluster_name}
        Should Be Equal As Strings    ${status}    True    msg=Cluster ${cluster_name} not deleted



*** keywords ***
SSH into Invader and Verify mass installation started
        [Arguments]    ${invader_ip}
        Open Connection     ${invader_ip}    timeout=1 hour
        Login               ${invader_usr_name}        ${invader_usr_pwd}
        Sleep    2s
        ${output}=         Execute Command    ps -aef | grep ROOT
        Log    \n\n INVADER DATA = ${output}    console=yes
        Should Contain    ${output}    tinyproxy.conf
        Close All Connections

Verify CentOS installed in server machine
        # Get OS release data from server
        ${rc}  ${output}    Run And Return Rc And Output        ssh ${server_node_host} "cat /etc/os-release"
        Log    \n\nGET OS release status Code = ${rc}    console=yes
        Log    \n\nSERVER OS RELEASE DATA = ${output}    console=yes
        Should Contain    ${output}    CentOS Linux

        ${rc}  ${output}    Run And Return Rc And Output        ssh ${server_node_host} "uptime -p"
        Log    \n\nServer uptime status code = ${rc}    console=yes
        Log    \n\nSERVER UP Time Data DATA = ${output}    console=yes
        ${status}    Verify server up time     ${output}
        Should Be Equal As Strings    ${status}    True    msg=There are no new OS deployed in last few minutes


*** Variables ***
${remove_force}    {"forceRemove":false}
