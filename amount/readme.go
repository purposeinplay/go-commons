// Package amount implements an Money type used to represent
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
// It also implements a ValueSubunit type that is used to store,
// persist and represent large monetary amounts in their
// lowest denominator form.
package amount
