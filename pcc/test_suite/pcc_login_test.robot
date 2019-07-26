*** Settings ***
Library  	OperatingSystem
Library  	Collections
Library  	String

Library    	${CURDIR}/../lib/Request.py
Variables       ${CURDIR}/../test_data/Login_Test_Data.py
Variables       ${CURDIR}/../test_data/Url_Paths.py
Library         ${CURDIR}/../lib/Data_Parser.py
Resource        ${CURDIR}/../resource/Resource_Keywords.robot


*** Test Cases ***

PCC-login-page-testing-with-valid-credentials
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with valid username, password and url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=${valid_user_name}    password=${valid_user_pwd}
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  200
        ${bearer_token}    Catenate    Bearer    ${resp.json()['token']}
        Set Suite Variable    ${sec_token}    ${bearer_token}
        &{auth_header}    Create Dictionary   Authorization=${sec_token}
        Set Suite Variable    ${headers}    ${auth_header}


PCC-login-page-testing-with-invalid-username
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with invalid username,valid password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=${invalid_user_name}    password=${valid_user_pwd}
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!


PCC-login-page-testing-with-invalid-password
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with valid username, invalid password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=${valid_user_name}    password=${invalid_user_pwd}
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!


PCC-login-page-testing-with-invalid-credentials
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with invalid username,invalid password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=${invalid_user_name}    password=${invalid_user_pwd}
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!


PCC-login-page-testing-with-empty-username
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with empty username,valid password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=    password=${valid_user_pwd}
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!


PCC-login-page-testing-with-empty-password
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with valid username,empty password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=${valid_user_name}    password=
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!


PCC-login-page-testing-with-empty-usernameandpassword
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with empty username,empty password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=    password=
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!


PCC-login-page-testing-with-empty-username-and-invald-password
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with empty username, invalid password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=    password=${invalid_user_pwd}
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!


PCC-login-page-testing-with-invalid-username-and-empty-password
        [Tags]    Login    regression_test
        [Documentation]    Test login page of PCC UI with invalid username, empty password and valid url
        [Setup]    Create Session    platina    ${server_url}    verify=False
        [Teardown]    Delete All Sessions

        &{data}=    Create Dictionary   username=${invalid_user_name}    password=
        ${resp}  Post Request    platina   ${login}    json=${data}
        Log    \n Status Code = ${resp.status_code}    console=yes
        Log    \n Response Data = ${resp.json()}    console=yes
        Should Be Equal As Strings  ${resp.status_code}  401
        Should Contain    ${resp.json()}    Bad credentials!