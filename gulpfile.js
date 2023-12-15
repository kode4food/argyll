"use strict";

var fs = require('fs');

var gulp = require('gulp');
var jshint = require('gulp-jshint');
var nodeunit = require('gulp-nodeunit');
var istanbul = require('gulp-istanbul');
var enforcer = require('gulp-istanbul-enforcer');

var plato = require('plato');

var pkg = require('./package.json');

var reportDir = './report';
var ignoreFiles = ['!coverage/**/*', '!report/**/*', '!node_modules/**/*'];
var sourceFiles = ['**/*.js'].concat(ignoreFiles);
var reportFiles = ['*.js', 'lib/**/*.js', 'test/**/*.js'];
var testFiles = ['test/*.js'];

var nodeUnitConfig = {
  reporter: 'default',
  reporterOptions: {
    output: 'test'
  }
};

var platoConfig = {
  title: "Complexity: " + pkg.name,
  jshint: {
    options: JSON.parse(fs.readFileSync('./.jshintrc').toString())
  },
  recurse: true
};

var enforcerConfig = {
  thresholds: {
    statements: 90,
    branches: 90,
    lines: 90,
    functions: 90
  },
  coverageDirectory: 'coverage',
  rootDirectory: ''
};

function createUnitTests() {
  return gulp.src(testFiles).pipe(nodeunit(nodeUnitConfig));
}

gulp.task('lint', function (done) {
  gulp.src(sourceFiles)
      .pipe(jshint())
      .pipe(jshint.reporter('default'))
      .on('end', done);
});

gulp.task('test', function (done) {
  createUnitTests().on('end', done);
});

gulp.task('coverage', function (done) {
  gulp.src(sourceFiles)
      .pipe(istanbul())
      .pipe(istanbul.hookRequire())
      .on('finish', function () {
        createUnitTests().pipe(istanbul.writeReports()).on('end', done);
      });
});

gulp.task('enforce', ['lint', 'coverage'], function (done) {
  gulp.src('.')
      .pipe(enforcer(enforcerConfig))
      .on('end', done);
});

gulp.task('complexity', function (done) {
  plato.inspect(reportFiles, reportDir, platoConfig, function () {
    done();
  });
});

gulp.task('build', ['enforce', 'complexity']);
gulp.task('default', ['build']);
