var utils = require('utils');
var system = require('system');

var arg = casper.cli.get('arg');

var index = arg.indexOf(",");  // Gets 1st index
var file = arg.substr(index + 1); // Get element 1 (filename)

var url_base = arg.substr(0, index); // Get element 0 (url)
var url_orig = url_base.replace("0.0.0.0:8000", "kubevirt.io/user-guide");

var index = url_base.indexOf("#");  // Get 2nd index
var url = url_base.substr(0, index); // Get element 0
var selector = url_base.substr(index + 1);  // Get element 1
var timeoutLength = 2000;


casper.test.begin('file: ' + file + ', url: ' + url_orig, 1, function suite(test) {
    casper.start(url, function() {
        test.assertExists('#' + selector, "Find selector: #" + selector);
    }).run(function() {
        test.done();
    });
});
