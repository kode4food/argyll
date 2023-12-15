# Changes

## Version 1.3 - Removing Bower and Browserify Support
In the vast majority of cases, JavaScript code is delivered to a browser in a minified state.  This generally means that the argument names have been minified as well.  As such, the utility of Argyll is almost nil in the browser environment and so it makes very little sense to maintain browser support.  As of this version, Argyll will target node and io.js only.
