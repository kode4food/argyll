/*
 * Argyll (Asynchronous Result Gathering)
 * Licensed under the MIT License
 * see LICENSE.md
 *
 * @author Thomas S. Bradford (kode4food.it)
 */

"use strict";

var nodeunit = require('nodeunit');
var argyll = require('../index');

var testValueHandlerCount = 0;

/* istanbul ignore next */
function noOp() {
}

function testValueHandler(arg, test) {
  test.ok(++testValueHandlerCount <= 2);
}

function createCompletionReporter(test, itemList) {
  return report;

  function report(item) {
    var idx = itemList.indexOf(item);
    test.ok(idx !== -1);
    /* istanbul ignore if */
    if ( idx === - 1 ) {
      console.log("Reported item not found: " + item.toString());
      return;
    }
    itemList.splice(idx, 1);
    if ( !itemList.length ) {
      test.done();
    }
  }
}

exports["Argyll Tests"] = nodeunit.testCase({
  "Normal Functionality": function (test) {
    var report = createCompletionReporter(test, [
      acceptArgOnly, lateStart, lateComplete, handleError, done
    ]);

    var context = argyll.withContextName("arg")
                        .always(acceptArgOnly).throwing("err")
                        .maybe(handleError)
                        .always(testValueHandler)
                        .complete(done) // duplicated on purpose
                        .complete(done);

    function acceptArgOnly(arg) {
      test.equal(arg, context);
      test.equal(arg.getContextName(), "arg");
      arg.start(lateStart);
      process.nextTick(function () {
        arg.complete(lateComplete);
      });
      arg.provide("test", test);
      report(acceptArgOnly);
      throw new Error("Some Error");
    }

    function lateStart(g) {
      test.equal(g, context);
      report(lateStart);
    }

    function lateComplete(g) {
      test.equal(g, context);
      report(lateComplete);
    }

    function handleError(arg, err) {
      // This should test as optional
      test.equal(arg, context);
      test.equal(err.message, "Some Error");
      report(handleError);
    }

    function done(g) {
      test.equal(g, context);
      report(done);
    }
  },

  "Explicitly Provided": function (test) {
    var report = createCompletionReporter(test, [
      acceptProvidedValues, done
    ]);

    var context = argyll("arg", {
      default1: "A Default Value",
      default2: "Another Default Value"
    });
    context.provide({
      provide1: "A provided value",
      provide2: "Another provided value"
    });
    context.provide("provide3", "Yet another provided value");
    context.always(acceptProvidedValues);
    context.always(testValueHandler);
    context.complete(done);

    function acceptProvidedValues(arg, default1, provide1, default2,
                                  provide2, provide3) {
      test.equal(context, arg);
      test.equal(default1, "A Default Value");
      test.equal(default2, "Another Default Value");
      test.equal(provide1, "A provided value");
      test.equal(provide2, "Another provided value");
      test.equal(provide3, "Yet another provided value");
      arg.provide("test", test);
      report(acceptProvidedValues);
    }

    function done(g) {
      report(done);
    }
  },

  "Parent-Child Functionality": function (test) {
    var report = createCompletionReporter(test, [
      kickoff, acceptChild, implicitArg, acceptArgOnly, parentDone,
      bothArgs, gotEverything, testCallback
    ]);

    var parentContext = argyll.always(acceptChild)
                              .complete(parentDone);

    var childContext = argyll();
    childContext.start(kickoff);
    childContext.always(implicitArg);
    childContext.always(bothArgs).returning('logs');
    childContext.always(gotEverything);
    childContext.complete(parentContext.receives('child'));

    function implicitArg(g, user) {
      var data = g.data();
      test.equal(data.user, user);
      report(implicitArg);
    }

    function bothArgs(g, user, profile) {
      var data = g.data();
      test.equal(data.profile, profile);
      test.equal(data.user, user);
      test.equal(data.user.name, 'Thom Bradford');
      test.equal(data.profile.title, 'JavaScript Developer');
      report(bothArgs);
      return { entries: ['this is a log'] };
    }

    function gotEverything(g, user, logs, profile) {
      var data = g.data();
      test.equal(data.profile, profile);
      test.equal(data.user, user);
      test.equal(data.logs, logs);
      test.equal(data.user.name, 'Thom Bradford');
      test.equal(data.profile.title, 'JavaScript Developer');
      test.equal(data.logs.entries[0], 'this is a log');
      report(gotEverything);
    }

    function kickoff(g) {
      var userDelay = Math.random() * 500;
      var profileDelay = Math.random() * 500;

      setTimeout(function () {
        g.provide('user', { name: 'Thom Bradford' });
      }, userDelay);

      setTimeout(function () {
        testCallback(g.gather('err', 'profile?'));
      }, profileDelay);

      report(kickoff);
    }

    function acceptChild(g, child) {
      test.equal(parentContext, g);
      test.equal(child, childContext.data());
      parentContext.always(acceptArgOnly);
      report(acceptChild);
    }

    function acceptArgOnly(g) {
      test.equal(parentContext, g);
      report(acceptArgOnly);
    }

    function parentDone(g) {
      test.equal(parentContext, g);
      report(parentDone);
    }

    function testCallback(callback) {
      callback(null, { title: 'JavaScript Developer' });
      report(testCallback);
    }
  },

  "No Arguments": function (test) {
    test.throws(function () {
      argyll.always();
    }, Error, "should explode if no arguments");
    test.done();
  },

  "Explicit Arguments": function (test) {
    var report = createCompletionReporter(test, [
      begin, handleAllArguments, done
    ]);

    argyll.start(begin)
          .always(handleAllArguments, "arg1", "arg2", "arg3")
          .complete(done);

    function begin(g) {
      report(begin);
      g.provide("arg1", "this is arg1");
      g.provide("arg2", "this is arg2");
      g.provide("arg2", "arg2 should not be this value");
      g.provide("arg3", "this is arg3");
    }

    function handleAllArguments(a1, a2, a3) {
      test.equal(a1, "this is arg1");
      test.equal(a2, "this is arg2");
      test.equal(a3, "this is arg3");
      report(handleAllArguments);
    }

    function done() {
      report(done);
    }
  },

  "Required Arguments": function (test) {
    var report = createCompletionReporter(test, [begin, done]);

    argyll.requires("arg1", "arg2", "arg3")
          .start(begin)
          .complete(done);

    function begin(g) {
      report(begin);
      g.provide("arg1", "this is arg1");
      g.provide("arg2", "this is arg2");
      g.provide("arg2", "arg2 should not be this value");
      g.provide("arg3", "this is arg3");
    }

    function done(g) {
      var data = g.data();
      test.equal(data.arg1, "this is arg1");
      test.equal(data.arg2, "this is arg2");
      test.equal(data.arg3, "this is arg3");
      report(done);
    }
  },

  "No Callback": function (test) {
    test.throws(function () {
      argyll.maybe("I shouldn't be here");
    }, Error, "should explode if no callback");
    test.done();
  },

  "Bad Result Names": function (test) {
    test.throws(function () {
      argyll.always(noOp, 99, 'result', 'other result');
    }, Error, "should explode if bad resultNames");
    test.done();
  },

  "Bad Provides": function (test) {
    test.throws(function () {
      argyll.provide(99, "splosion");
    }, Error, "should explode if bad resultName");
    test.done();
  },

  "Bad Context Name": function (test) {
    test.throws(function () {
      argyll.withContextName(99);
    }, Error, "should explode if bad contextName");
    test.done();
  },

  "Returning Undefined": function (test) {
    var g = argyll().always(test1).returning('next')
                    .always(test2).returning('next')
                    .always(test3);

    g.provide('start1', 'start1 value');
    setTimeout(function () {
      g.provide('start2', 'start2 value');
    }, 100);

    function test1(start1) {
      test.equal(start1, 'start1 value');
    }

    function test2(start2) {
      test.equal(start2, 'start2 value');
      return 'returned value';
    }

    function test3(next) {
      test.equal(next, 'returned value');
      test.done();
    }
  },

  "Good Gathered Arguments": function (test) {
    var report = createCompletionReporter(test, [alwaysTest, maybeTest, done]);

    var g = argyll().always(alwaysTest).maybe(maybeTest).complete(done);
    var cb1 = g.gather("err!", "name?");
    var cb2 = g.gather("err!", "name", "bug");

    cb2(new Error("First Error"), "invalid name", "invalid bug");
    cb1(new Error("Second Error"), null);
    cb1(null, "the first name");
    cb2(null, "the second name");
    cb2(null, "the third name", "the bug");

    function alwaysTest(name, bug) {
      test.equal(name, "the first name");
      test.equal(bug, "the bug");
      report(alwaysTest);
    }

    function maybeTest(err) {
      test.ok(err);
      test.equal(err.message, "First Error");
      report(maybeTest);
    }

    function done(g) {
      report(done);
    }
  },

  "Bad Gathered Arguments": function (test) {
    test.throws(function () {
      var g = argyll();
      g.gather("incorrect  ?");
    }, Error, "should explode if bad gather argument");
    test.done();
  },

  "Cancel Works": function (test) {
    var startCalled = false;
    var alwaysCalled = false;
    var doneCalled = false;

    var g = argyll().start(startTest).always(alwaysTest).complete(done);
    g.provide("test", "some value");
    g.cancel();

    setTimeout(function () {
      test.ok(!startCalled);
      test.ok(!alwaysCalled);
      test.ok(!doneCalled);
      test.done();
    }, 100);

    /* istanbul ignore next */
    function startTest() {
      startCalled = true;
    }

    /* istanbul ignore next */
    function alwaysTest(test) {
      alwaysCalled = true;
    }

    /* istanbul ignore next */
    function done() {
      doneCalled = true;
    }
  },

  "Many Events At Once": function (test) {
    test.expect(100);

    var g = argyll();

    for ( var i = 0; i < 100; i++ ) {
      fireHandler(i);
    }

    g.complete(function () {
      test.done();
    });

    function fireHandler(waitFor) {
      var strVal = '' + waitFor;
      g.always(handler, strVal);
      g.provide(strVal, waitFor);

      function handler(value) {
        test.equal(value, waitFor);
      }
    }
  },

  "Long Chain of Events": function (test) {
    test.expect(101);

    var g = argyll();
    fireHandler(0);
    g.complete(function () {
      test.done();
    });

    var waitingFor;

    function fireHandler(value) {
      var strVal = '' + value;
      g.always(handler, strVal);
      g.provide(strVal, value);
      waitingFor = value;
    }

    function handler(value) {
      test.equal(value, waitingFor);
      if ( value < 100 ) {
        fireHandler(value + 1);
      }
    }
  }
});
