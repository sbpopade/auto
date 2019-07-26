import requests
import json





#resp = requests.get("https://172.17.2.46:7654/setPass/token=eyJhbGciOiJIUzUxMiJ9.eyJzdWIiOiJhdXRvX3Rlc3QyIiwiZXhwIjoxNTYxNzIxODU2LCJpYXQiOjE1NjExMTcwNTZ9.ty-SyIVUV1SfgSV7hpAlKz2YJgvPxrlbL669w2KNNHdfjPn6CE6i4BQFLucggRin-6CJxQdoXNYYAiq5e_iScg", verify = False)

#print(resp.status_code)






test_data = {'username':'admin', 'password':'admin'}

# Login
resp = requests.post('https://172.17.2.47:9999/security/auth/', json=test_data, verify = False)
print(resp.status_code)
print(resp.json())
token = 'Bearer ' + resp.json()['token']
print(token)
headers = {'Authorization': token,}
#
#
#data = {"username":"auto_test","firstname":"auto","lastname":"test","email":"platina.automation@gmail.com","active":"true", "tenant":"1", "roleID":"1"}
#
#
##resp = requests.post("https://172.17.2.46:9999/user-management/user/register", json=data, verify = False, headers = headers)
##print(resp.status_code)
##print(resp.json())



data = {"password":"platina123"}
resp = requests.post("https://172.17.2.46:9999/user-management/user/set-password", json=data, verify = False)
print(resp.status_code)
print("pwd updated successfully...")






#data = {"Id":1, "Site_Id":29}
#resp = requests.put('https://172.17.2.46:9999/pccserver/node/update', json=data, verify = False, headers = headers)
#print(resp.status_code)
#print(resp.json()
## Get node roles
#headers = {
#    'Authorization': token,
#}
#resp = requests.get('https://172.17.2.46:9999/pccserver/roles/', headers = headers, verify = False)
#print(resp.status_code)
#print(resp.json())
#
#
# add Node
#data = {"Name":"i30","Host":"172.17.2.29", }
#data = {"Name":"test", "Host":"172.17.2.29", "bmc":"172.17.3.29", "bmcUser":"ADMIN", "bmcUsers":["ADMIN"], "bmcPassword":"ADMIN"}
#resp = requests.post('https://172.17.2.46:9999/pccserver/node/add', headers = headers, json=data, verify = False)
#print(resp.status_code)
#print(resp.json())

## Get Added Node 
#
#params = {'page':0, 'limit':50, 'sortBy':'name', 'sortDir':'asc', 'search':''}
#resp = requests.get('https://172.17.2.46:9999/pccserver/node', headers = headers, verify = False, params=params)
#print(resp.status_code)
#print(resp.json())
#
#
# params = {"Id":0,"Name":"site_1","Description":"site_1"}
