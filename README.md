# Argyll (Asynchronous Result Gathering)
[![Build Status](https://travis-ci.org/kode4food/argyll.svg)](https://travis-ci.org/kode4food/argyll)

Argyll is a simple little library that allows you to parallelize the process of gathering asynchronous results.  I wrote it because I was performing complex workflows using Promises that didn't truly need to be serialized.  Where this library is most useful is enabling you to perform multiple parallel calls, only triggering callbacks when the callback's result dependencies are satisfied.

A basic bogus skeletal example:

```javascript
var Argyll = require('argyll');

Argyll.start(makingRequests)
      .always(acceptEverything)
      .always(acceptTemplate)
      .maybe(handleError);

function makingRequests(g) {
  // Make some calls all at once, the g.gather() function
  // creates an intercept proxy that routes the responses
  // appropriately
  User.findById(someUserId, g.gather('err!', 'user?'));
  Profile.findById(someProfileId, g.gather('err!', 'profile?'));
  Logs.findAll(someUserId, g.gather('err!', 'logs'));
}

function acceptEverything(g, user, profile, logs) {
  // We'll only get here when all three results have been
  // fully resolved, even though they were resolved using
  // three separate asynchronous calls
  var result = processTemplate(g.data());
  
  // 'provide' will allow you to explicitly satisfy a 
  // result dependency. 'template' is the result name
  // that will be fulfilled
  g.provide('template', result);

  // We could have also written:
  //     return processTemplate(g.data());
  // and then just registered this callback using:
  //     always(acceptEverything).returning('template')
}

function acceptTemplate(template) {
  // You'll notice this function doesn't take a 'g' 
  // argument and it still works perfectly fine.
  res.end(template);
}

function handleError(g, err) {
  // Uh-oh!  Ignore any subsequent calls
  g.cancel();
}
```

You'll notice that the creation of the gathering chain didn't specify which results the functions were in charge of gathering.  This is because the values are extracted from the Function's signature.  All of the argument names are treated as the result names to gather.  You can also specify them explicitly if you like:

```javascript
Argyll.start(makingRequests)
      .always(acceptEverything, 'g', 'user', 'profile', 'logs')
      .always(acceptTemplate, 'template')
      .maybe(handleError, 'g', 'err');
```

What you may have *not* noticed is that the order in which you register callbacks actually doesn't matter much.  Argyll will invoke a callback as soon as all of its dependencies have been satisfied, regardless of its order in the gathering context's definition, and regardless of whether or not a callback defined earlier has been triggered.

## The API
The value returned by `require('argyll')` is a function that can be used to create a gathering context.  It also exposes some immediate child functions for kicking things off.  They are `always()`, `maybe()`, `requires()` and `start()`.  These functions behave exactly as the ones exposed by the gathering context except that they also *create* a context.

Another function `withContextName({String} contextName)` sets the name of the Argyll gathering context.  This is the result name as it will be provided to callbacks that accept it.  By default, it is 'g'.  Synonyms include `setContextName()`.  This function can only be used to kick things off.

`argyll([{String} contextName], [{Object} defaults])` - creates and returns a new gathering context.  The previous chain could have been written as follows, making the separate makingRequests function unnecessary:

```javascript
var argyll = require('argyll');
var g = argyll();
g.always(acceptEverything);
g.always(acceptTemplate);
g.maybe(handleError);

User.findById(someUserId, g.gather('err!', 'user?'));
Profile.findById(someProfileId, g.gather('err!', 'profile?'));
Logs.findAll(someUserId, g.gather('err!', 'logs'));
```

The gathering context contains the following functions:

`data()` - returns an Object containing the results collected thus far by the gathering context.

`start({Function} callback)` - Register a callback to be used to kick things off.  This callback will only take a reference to the gathering context.  Synonyms include `onStart()` and `startsWith()`.

`always({Function} callback, {String*} resultNames)` - Registers a callback to be fired when its dependent results have been resolved.  If no resultNames are specified, they will be extracted from the callback Function arguments. Results gathered with this callback are considered necessary in reporting completion.  Synonyms include `must()`.

`maybe({Function} callback, {String*} resultNames)` - Registers a callback to be fired when its dependent results have been resolved.  If no resultNames are specified, they will be extracted from the callback Function arguments.  Results gathered with this callback are considered optional in reporting completion, making it good for error handlers.  Synonyms include `sometimes()`.
                                                        
`complete({Function} callback)` - Register a callback to be called when the gathering context has no pending required results to gather.  Synonyms include `onComplete()`, `whenComplete()` and `completesWith()`.

`requires({String*} resultNames)` - Instructs the gathering context that the specified resultNames are required to trigger a completion callback, even if no explicit interest is registered by other functions.  Synonyms include `require()`.

`provide({String} resultName, {Mixed} value)` - Explicitly provide a named result value.

`cancel()` - Cancels all pending callbacks.  Useful if you want to short-circuit the process upon an error.

`gather({String*} resultNames)` - Generates a node-compatible callback that can be used to automatically provide results (including errors).  If the result names are suffixed with a question mark (`?`) then `null` values will be accepted (`undefined` will *never* be accepted).  If the result names are suffixed with an exclamation mark (`!`), then a provided result will short-circuit any subsequent argument processing.  Synonyms include `callback()`.

`receives({String} resultName)` - Generates a 'receives' callback that can be passed to the `complete()` method of another gathering context.  When that context has completed gathering all of its data, it will provide that data to the specified resultName of this gathering context.

`contextName()` - Returns the name of the Argyll gather context as it will be provided to callbacks that accept it.  By default, it is 'g'.  Synonyms include `getContextName()`.

`returning({String} resultName)` - Can be applied to `always()` and `maybe()` registrations.  This tells Argyll that the callback returns a value that provides the specified resultName.  Synonyms include `returns()`.

`throwing({String} resultName)` - Can be applied to `always()` and `maybe()` registrations.  This tells Argyll that the callback may throw an Exception, the value of which provides the specified resultName.  Synonyms include `throws()`.

## Installing
To install from NPM (Node Package Manager), type:

```bash
npm install argyll
```

## Resources
For the latest releases, see the [Argyll GitHub Page](http://github.com/kode4food/argyll)

## License (MIT License)
Copyright (c) 2014, 2015 Thomas S. Bradford

Permission is hereby granted, free of charge, to any person
obtaining a copy of this software and associated documentation
files (the "Software"), to deal in the Software without
restriction, including without limitation the rights to use,
copy, modify, merge, publish, distribute, sublicense, and/or
sell copies of the Software, and to permit persons to whom the
Software is furnished to do so, subject to the following
conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.
