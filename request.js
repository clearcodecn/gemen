(function () {
    return {
        request: {
            // urlMatch
            urlMatch: function (url) {
                return url.lastIndexOf("zhihu.com") !== -1
            },
            methodMatch: function (method) {
                return method === "GET" || method === "POST"
            },
            beforeRequest: function (request) {
                return request
            }
        },
        response: {
            contentTypeMatch: function (contentType) {
                return true
            },
            onResponse: function (request, response) {
                return response
            }
        }
    }
})()