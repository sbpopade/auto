*** Keywords ***
Verify User Login
	Create Session    platina    ${server_url}    verify=False
    	&{data}=    Create Dictionary   username=${user_name}    password=${user_pwd}
	${resp}  Post Request    platina   ${login}    json=${data}
    	Should Be Equal As Strings  ${resp.status_code}  200
    	${bearer_token}    Catenate    Bearer    ${resp.json()['token']}
	Set Suite Variable    ${sec_token}    ${bearer_token}
	&{auth_header}    Create Dictionary   Authorization=${sec_token}
	Set Suite Variable    ${headers}    ${auth_header}
