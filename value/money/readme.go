// Package money implements an Amount type used to represent
// a monetary amount defined by the following properties:
//
// - value, in the lowest denominator form, eg. cents for USD.
//
// - decimals, the number of the digits after
// the decimals point, eg. 2 for USD.
//
// - currency code, the shorthand for
// the currency, eg. USD for United States Dollar.
//
// The package is placed under the ./value package
// as it imports from it.
package money
