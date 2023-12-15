/*
 * Argyll (Asynchronous Result Gathering)
 * Licensed under the MIT License
 * see LICENSE.md
 *
 * @author Thomas S. Bradford (kode4food.it)
 */

"use strict";

/* istanbul ignore next */
var nextTick = (function () {
  if ( typeof setImmediate === 'function' ) {
    return setImmediate;
  }
  if ( typeof window === 'object' &&
    typeof window.requestAnimationFrame === 'function' ) {
    if ( typeof window.requestAnimationFrame.bind === 'function' ) {
      return window.requestAnimationFrame.bind(window);
    }
    return function () {
      window.requestAnimationFrame.apply(window, arguments);
    };
  }
  if ( typeof setTimeout === 'function' ) {
    return setTimeout;
  }
  throw new Error("And I should schedule tasks how?");
}());

function Scheduler(idleHandler) {
  this._idleHandler = idleHandler;
  this._capacity = 16 * 2;
  this._isFlushing = false;
  this._queueIndex = 0;
  this._queueLength = 0;
}

Scheduler.prototype.queue = function (callback) {
  this[this._queueLength++] = callback;
  if ( !this._isFlushing ) {
    this._isFlushing = true;
    var self = this;
    nextTick(function () {
      self.flushQueue();
    });
  }
};

Scheduler.prototype.resetQueue = function () {
  for ( var i = 0, len = this._queueLength; i < len; i++ ) {
    this[i] = undefined;
  }

  this._isFlushing = false;
  this._queueIndex = 0;
  this._queueLength = 0;
};

Scheduler.prototype.collapseQueue = function () {
  var queueIndex = this._queueIndex;
  var queueLength = this._queueLength;
  var i = 0;
  var len = queueLength - queueIndex;
  for ( ; i < len; i++ ) {
    this[i] = this[queueIndex + i];
  }
  while ( i < queueLength ) {
    this[i++] = undefined;
  }
  this._queueIndex = 0;
  this._queueLength = len;
};

Scheduler.prototype.flushQueue = function () {
  while ( this._queueIndex < this._queueLength ) {
    var callback = this[this._queueIndex++];

    if ( this._queueLength > this._capacity ) {
      this.collapseQueue();
    }

    callback();
  }
  this._isFlushing = false;
  this._idleHandler();
};

exports.Scheduler = Scheduler;
