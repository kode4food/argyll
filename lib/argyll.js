/*
 * Argyll (Asynchronous Result Gathering)
 * Licensed under the MIT License
 * see LICENSE.md
 *
 * @author Thomas S. Bradford (kode4food.it)
 */

"use strict";

var byname = require('byname');
var Scheduler = require('./scheduler').Scheduler;

var objectKeys = Object.keys;
var extendObject = Object.create;
var slice = Array.prototype.slice;

function noOp() {}

var gatherRegex = /^\s*([^?!\s]+)([?!]?)\s*$/;
var gatherArgCache = {};

/**
 * Constructs a new gathering context
 */
function argyll(contextName, defaults) {
  if ( typeof contextName !== 'string' ) {
    defaults = contextName;
    contextName = 'g';
  }

  // `gathered` stores the results that have already been gathered
  // `interest` tracks result names to all interested entries
  // `requiredResults` tracks result names that are required in this context
  // `starters` tracks who to call before any callbacks are fired
  // `finishers` tracks who to call when no interest remains
  // `started` is set to true if the initial starters have been fired

  var gathered = {};         // resultName -> value
  var interest = {};         // resultName -> [entry]
  var requiredResults = {};  // resultName -> true
  var starters = [];
  var finishers = [];
  var started;

  var scheduler = new Scheduler(checkComplete);

  var context = {
    data: data,
    start: start, onStart: start, startWith: start,
    must: always, should: always, always: always,
    maybe: maybe, sometimes: maybe,
    complete: complete, onComplete: complete,
    whenComplete: complete, completesWith: complete,
    requires: requires, require: requires,
    provide: provide,
    cancel: cancel,
    gather: gather, callback: gather,
    receives: receives,
    contextName: getContextName, getContextName: getContextName
  };

  queueCall(kickoff);
  return context;

  // External Interface *******************************************************

  // Kick things off for the context.  This is done by first calling any
  // registered 'starters', then by providing the context as a result, and
  // then providing the defaults object (if any)
  function kickoff() {
    for ( var i = 0, len = starters.length; i < len; i++ ) {
      queueCall(starters[i], [context]);
    }
    starters = [];
    provide(contextName, context);
    if ( typeof defaults === 'object' && defaults !== null ) {
      provideObject(defaults);
      defaults = null;
    }
    started = true;
  }

  // guess what this does?
  function getContextName() {
    return contextName;
  }

  // returns an Object containing the results collected thus far by the
  // gathering context
  function data() {
    return gathered;
  }

  // Register a callback to be used to kick things off.  This callback will
  // only take a reference to the gathering context
  function start(callback) {
    if ( started ) {
      queueCall(callback, [context]);
      return context;
    }
    starters.push(callback);
    return context;
  }

  // Registers a callback to be fired when its dependent results have been
  // resolved.  If no resultNames are specified, they will be extracted from
  // the callback Function arguments.  Results gathered with this callback
  // are considered necessary in reporting completion
  function always(callback) {
    return createEntry(arguments, true);
  }

  // Registers a callback to be fired when its dependent results have been
  // resolved.  If no resultNames are specified, they will be extracted from
  // the callback Function arguments.  Results gathered with this callback
  // are considered optional in reporting completion, making it good for
  // error handlers
  function maybe(callback) {
    return createEntry(arguments, false);
  }

  // Register a callback to be called when the gathering context has no
  // pending required results to gather
  function complete(callback) {
    if ( finishers.indexOf(callback) !== -1 ) {
      // If it's already registered, we don't register it again
      return context;
    }
    finishers.push(callback);
    if ( started ) {
      // Make sure the call queue wakes up long enough to check for completion
      queueCall(noOp);
    }
    return context;
  }

  // Iterate over the pairs in an object to provide named results
  function provideObject(obj) {
    var keys = objectKeys(obj);
    for ( var i = 0, len = keys.length; i < len; i++ ) {
      var key = keys[i];
      provide(key, obj[key]);
    }
  }

  // Explicitly require a set of named results
  function requires() {
    var argArray = slice.call(arguments, 0);
    createEntry([noOp].concat(argArray), true);
    return context;
  }

  // Explicitly provide a named result value
  function provide(resultName, value) {
    var resultNameType = typeof resultName;
    if ( resultNameType !== 'string' ) {
      if ( resultNameType === 'object' && resultName !== null ) {
        provideObject(resultName);
        return context;
      }
      throw new Error("resultName must be a string: " + resultName);
    }

    // If the value has already been gathered, we don't corrupt the context's
    // state by overriding it
    if ( value === undefined || gathered[resultName] !== undefined ) {
      return context;
    }

    gathered[resultName] = value;

    // We can clear this list since any further calls to `always()` will find
    // the resultName already fulfilled
    var entries = interest[resultName];
    if ( entries ) {
      delete interest[resultName];

      for ( var i = 0, len = entries.length; i < len; i++ ) {
        var entry = entries[i];
        if ( canPerformCallback(entry) ) {
          // Queue up the callback, all args were resolved
          queueCall(createEntryCall(entry), [gathered]);
        }
      }
    }

    return context;
  }

  // Cancels all pending callbacks.  Useful if you want to short-circuit the
  // process upon an error
  function cancel() {
    starters = [];
    interest = {};
    finishers = [];
    scheduler.resetQueue();
    return context;
  }

  function getGatherArgument(arg) {
    var result = gatherArgCache[arg];
    if ( result ) {
      return result;
    }

    var match = gatherRegex.exec(arg);
    if ( !match ) {
      throw new Error("Callback argument invalid: " + arg);
    }

    result = gatherArgCache[arg] = {
      name: match[1],
      action: match[2]
    };
    return result;
  }

  // Generates a node-compatible callback that can be used to automatically
  // provide results (including errors).  If the result names are suffixed
  // with a question mark (`?`) then `null` values will be accepted
  // (`undefined` will never be accepted).  If the result names are suffixed
  // with an exclamation mark (`!`), then a provided result will short-circuit
  // any subsequent argument processing
  function gather() {
    var callbackArgs = [];
    for ( var i = 0, len = arguments.length; i < len; i++ ) {
      callbackArgs.push(getGatherArgument(arguments[i]));
    }
    return resultCallback;

    function resultCallback() {
      for ( var i = 0, len = callbackArgs.length; i < len; i++ ) {
        var val = arguments[i];
        if ( val === undefined ) {
          continue;
        }

        var callbackArg = callbackArgs[i];
        var action = callbackArg.action;
        if ( val !== null || action === '?' ) {
          provide(callbackArg.name, val);
          if ( action === '!' ) {
            return;
          }
        }
      }
    }
  }

  // Generates a 'receives' callback that can be passed to the `complete()`
  // method of another gathering context.  When that context has completed
  // gathering all of its data, it will provide that data to the specified
  // resultName of this gathering context
  function receives(resultName) {
    return receivesCallback;

    function receivesCallback(g) {
      provide(resultName, g.data());
    }
  }

  // Support Functions ********************************************************

  function createEntry(args, required) {
    var callback = byname.wrap.apply(null, args);
    var resultNames = callback.getArgumentNames();
    resultNames.forEach(function (resultName) {
      if ( typeof resultName !== 'string' ) {
        throw new Error("resultNames must be Strings");
      }
    });

    var entry = [resultNames, callback];
    if ( !registerEntry(entry, required) ) {
      // If we can't register the entry (because all result dependencies have
      // been fulfilled) then we just queue it up

      /* istanbul ignore if */
      if ( !canPerformCallback(entry) ) {
        throw new Error("Argyll is broken! All arguments should be available");
      }

      queueCall(createEntryCall(entry), [gathered]);
    }

    var entryContext = extendObject(context);
    entryContext.returns = entryContext.returning = returning;
    entryContext.throws = entryContext.throwing = throwing;

    return entryContext;

    function returning(resultName) {
      var prevCallback = entry[1];
      entry[1] = returningCallback;
      return entryContext;

      function returningCallback() {
        /* jshint validthis:true */
        var result = prevCallback.apply(this, arguments);
        if ( result !== undefined ) {
          provide(resultName, result);
        }
        return result;
      }
    }

    function throwing(resultName) {
      var prevCallback = entry[1];
      entry[1] = throwingCallback;
      return entryContext;

      function throwingCallback() {
        try {
          /* jshint validthis:true */
          return prevCallback.apply(this, arguments);
        }
        catch ( err ) {
          provide(resultName, err);
        }
      }
    }
  }

  function createEntryCall(entry) {
    return entryCall;

    function entryCall() {
      /* jshint validthis:true */
      return entry[1].apply(this, arguments);
    }
  }

  function registerEntry(entry, required) {
    var resultNames = entry[0];
    var registered = false;

    for ( var i = 0, len = resultNames.length; i < len; i++ ) {
      var resultName = resultNames[i];
      if ( required ) {
        requiredResults[resultName] = true;
      }
      if ( gathered[resultName] !== undefined ) {
        continue;
      }
      var entries = interest[resultName] || (interest[resultName] = []);
      entries.push(entry);
      registered = true;
    }
    return registered;
  }

  function canPerformCallback(entry) {
    var resultNames = entry[0];
    for ( var i = 0, len = resultNames.length; i < len; i++ ) {
      var resultName = resultNames[i];
      if ( interest[resultName] ) {
        // We're still waiting for a result
        return false;
      }
    }
    return true;
  }

  function queueCall(callback, callArgs) {
    scheduler.queue(function () {
      callback.apply(null, callArgs);
    });
  }

  function checkComplete() {
    if ( !finishers.length ) {
      return;
    }

    for ( var resultName in interest ) {
      /* istanbul ignore else */
      if ( requiredResults[resultName] ) {
        // There's interest.  We're not done yet
        return;
      }
    }

    for ( var i = finishers.length - 1; i >= 0; i-- ) {
      queueCall(finishers[i], [context]);
    }
    finishers = [];
  }
}

function start() {
  /* jshint validthis:true */
  return argyll().start.apply(this, arguments);
}

function setContextName(contextName) {
  if ( typeof contextName !== 'string' ) {
    throw new Error("contextName must be a string");
  }
  return argyll(contextName);
}

function always() {
  /* jshint validthis:true */
  return argyll().always.apply(this, arguments);
}

function maybe() {
  /* jshint validthis:true */
  return argyll().maybe.apply(this, arguments);
}

function requires() {
  /* jshint validthis:true */
  return argyll().requires.apply(this, arguments);
}

function provide() {
  /* jshint validthis:true */
  return argyll().provide.apply(this, arguments);
}

// Exports
argyll.withContextName = argyll.setContextName = setContextName;
argyll.must = argyll.should = argyll.always = always;
argyll.sometimes = argyll.maybe = maybe;
argyll.onStart = argyll.startsWith = argyll.start = start;
argyll.require = argyll.requires = requires;
argyll.provide = provide;

module.exports = argyll;
