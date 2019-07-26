from RequestsLibrary import RequestsLibrary


class Request(RequestsLibrary):
        def post_request(
            self,
            alias,
            uri,
            data=None,
            json=None,
            params=None,
            headers=None,
            files=None,
            allow_redirects=None,
            timeout=None):
	    """ Send a POST request on the session object found using the
	        given `alias`
	        ``alias`` that will be used to identify the Session object in the cache
	        ``uri`` to send the POST request to
	        ``data`` a dictionary of key-value pairs that will be urlencoded
	               and sent as POST data
	               or binary data that is sent as the raw body content
	               or passed as such for multipart form data if ``files`` is also
	                  defined
	        ``json`` a value that will be json encoded
	               and sent as POST data if files or data is not specified
	        ``params`` url parameters to append to the uri
	        ``headers`` a dictionary of headers to use with the request
	        ``files`` a dictionary of file names containing file data to POST to the server
	        ``allow_redirects`` Boolean. Set to True if POST/PUT/DELETE redirect following is allowed.
	        ``timeout`` connection timeout
	    """
            # import sys, pdb; pdb.Pdb(stdout=sys.__stdout__).set_trace()
	    session = self._cache.switch(alias)
	    if not files:
	        data = self._format_data_according_to_header(session, data, headers)
	    redir = True if allow_redirects is None else allow_redirects
            if data:
        	import json; data = json.loads(data)

            if json:
                if type(json) == list:
                    json = [ int(x) for x in json ]

            if str(uri) == "/maas/deployments/":
               json['nodes'] = eval(json['nodes'])
               json['sshKeys'] = eval(json['sshKeys'])

            response = self._body_request(
	            "post",
	            session,
	            uri,
	            data,
	            json,
	            params,
	            files,
	            headers,
	            redir,
	            timeout)
            dataStr = self._format_data_to_log_string_according_to_header(data, headers)
            return response

        def put_request(
            self,
            alias,
            uri,
            data=None,
            json=None,
            params=None,
            files=None,
            headers=None,
            allow_redirects=None,
            timeout=None):
            """ Send a PUT request on the session object found using the
            given `alias`
            ``alias`` that will be used to identify the Session object in the cache
            ``uri`` to send the PUT request to
            ``data`` a dictionary of key-value pairs that will be urlencoded
               and sent as PUT data
               or binary data that is sent as the raw body content
            ``json`` a value that will be json encoded
               and sent as PUT data if data is not specified
            ``headers`` a dictionary of headers to use with the request
            ``allow_redirects`` Boolean. Set to True if POST/PUT/DELETE redirect following is allowed.
            ``params`` url parameters to append to the uri
            ``timeout`` connection timeout
            """

            session = self._cache.switch(alias)
            data = self._format_data_according_to_header(session, data, headers)
            redir = True if allow_redirects is None else allow_redirects

            if json:
                try:
                    json = { str(key):int(val) for key, val in json.items()}
                except:
                    json= json
		if json.has_key("Id"):
                    json['Id'] = int(json['Id'])

            if json:
                if json.has_key("roles"):
                    if type(json["roles"]) == list:
                        json["roles"] = [ int(val) for val in json["roles"] ]
                    else:
                        json["roles"] = [json["roles"]]

            response = self._body_request(
            "put",
            session,
            uri,
            data,
            json,
            params,
            files,
            headers,
            redir,
            timeout)

            if isinstance(data, bytes):
                data = data.decode('utf-8')
            print('Put Request using : alias=%s, uri=%s, data=%s, \
                    headers=%s, allow_redirects=%s ' % (alias, uri, data, headers, redir))

            return response


#        def custom_post_request(self, url='', data=''):
#            """
#            Custom post request without session creation
#            """
#            import requests
#            import sys, pdb; pdb.Pdb(stdout=sys.__stdout__).set_trace()
#
#            resp = requests.post(url, json=data, verify = False)
#            print("resp.status_code")
#            print("resp.jaon()")
#            return {"status_code" : resp.status_code, "json" : resp.json()}
