//! spark: turn a list of numbers into a sparkline.
//!
//! The library is split into small stages so each can be tested alone:
//! parse the input, map values to levels, then render levels to text.

pub mod parse;
