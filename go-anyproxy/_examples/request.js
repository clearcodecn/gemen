const request = {
    url: new URL('../cats', 'http://www.example.com/dogs'),
    headers: {
        key: value,
    },
    cookies: "string",
    body: "",
}


function injectRequest(request) {
    // change request.
    // and return new headers, cookies,
    request.headers.token = "abc"
    return request
}

// response.statusCode.
// response.headers
// response.body
function injectResponse(response) {
    // change request.
    // and return new headers, cookies,
    header.token = "xyz"
    return {

    }
}