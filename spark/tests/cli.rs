//! End-to-end tests that run the real binary.

use std::io::Write;
use std::process::{Command, Output, Stdio};

fn spark(args: &[&str], stdin: Option<&str>) -> Output {
    let mut cmd = Command::new(env!("CARGO_BIN_EXE_spark"));
    cmd.args(args)
        .env_remove("LC_ALL")
        .env_remove("LC_CTYPE")
        .env("LANG", "en_US.UTF-8")
        .env_remove("NO_COLOR")
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    let mut child = cmd.spawn().expect("spawn spark");
    {
        let mut pipe = child.stdin.take().unwrap();
        if let Some(input) = stdin {
            pipe.write_all(input.as_bytes()).unwrap();
        }
    }
    child.wait_with_output().unwrap()
}

fn stdout(o: &Output) -> String {
    String::from_utf8(o.stdout.clone()).unwrap()
}

fn stderr(o: &Output) -> String {
    String::from_utf8(o.stderr.clone()).unwrap()
}

#[test]
fn numbers_as_arguments() {
    let o = spark(&["1", "5", "22", "13", "53"], None);
    assert!(o.status.success());
    assert_eq!(stdout(&o), "▁▁▄▂█\n");
}

#[test]
fn comma_separated_argument() {
    let o = spark(&["0,30,55,80,33,150"], None);
    assert_eq!(stdout(&o), "▁▂▃▅▂█\n");
}

#[test]
fn piped_input() {
    let o = spark(&[], Some("9\n13\n5\n17\n1\n"));
    assert!(o.status.success());
    assert_eq!(stdout(&o), "▅▇▃█▁\n");
}

#[test]
fn negative_numbers() {
    let o = spark(&["-5", "0", "5"], None);
    assert!(o.status.success(), "{}", stderr(&o));
    assert_eq!(stdout(&o), "▁▅█\n");
}

#[test]
fn leading_zeros() {
    let o = spark(&["007", "010", "1"], None);
    assert_eq!(stdout(&o), "▆█▁\n");
}

#[test]
fn gaps_and_stats() {
    let o = spark(&["--stats", "1", "x", "3", "9"], None);
    assert_eq!(
        stdout(&o),
        "▁ ▃█\nmin 1  max 9  avg 4.333  last 9  n 3  missing 1\n"
    );
}

#[test]
fn csv_holes_from_stdin() {
    let o = spark(&[], Some("1,,3\n"));
    assert_eq!(stdout(&o), "▁ █\n");
}

#[test]
fn ascii_from_locale_and_flag() {
    let mut cmd = Command::new(env!("CARGO_BIN_EXE_spark"));
    let o = cmd
        .args(["1", "2", "3", "4"])
        .env("LC_ALL", "C")
        .output()
        .unwrap();
    assert_eq!(stdout(&o), "_-+#\n");
    assert_eq!(stdout(&spark(&["--ascii", "1", "2"], None)), "_#\n");
}

#[test]
fn unicode_flag_overrides_locale() {
    let o = Command::new(env!("CARGO_BIN_EXE_spark"))
        .args(["--unicode", "1", "2"])
        .env("LC_ALL", "C")
        .output()
        .unwrap();
    assert_eq!(stdout(&o), "▁█\n");
}

#[test]
fn rows_and_width() {
    let input: String = (1..=40).map(|v| format!("{v}\n")).collect();
    let o = spark(&["-w", "10", "-r", "2"], Some(&input));
    assert_eq!(stdout(&o), "     ▁▃▅▇█\n▁▂▄▆██████\n");
}

#[test]
fn fixed_scale_and_log() {
    assert_eq!(
        stdout(&spark(&["--min", "0", "--max", "100", "50", "100"], None)),
        "▅█\n"
    );
    assert_eq!(
        stdout(&spark(&["--log", "1", "10", "100", "1000"], None)),
        "▁▃▆█\n"
    );
}

#[test]
fn color_always_and_no_color() {
    let o = spark(&["--color=always", "1", "2"], None);
    assert!(stdout(&o).contains("\x1b[32m"));
    // --color with no value means auto, and a pipe isn't a terminal.
    let o = spark(&["--color", "1", "2"], None);
    assert!(!stdout(&o).contains('\x1b'));
}

#[test]
fn errors_go_to_stderr_with_exit_codes() {
    let o = spark(&["1", "banana"], None);
    assert_eq!(o.status.code(), Some(1));
    assert!(stderr(&o).contains("banana"));
    assert!(stdout(&o).is_empty());

    let o = spark(&["--wat"], None);
    assert_eq!(o.status.code(), Some(2));

    let o = spark(&["--log", "0", "1"], None);
    assert_eq!(o.status.code(), Some(1));
    assert!(stderr(&o).contains("above zero"));

    let o = spark(&[], Some(""));
    assert_eq!(o.status.code(), Some(1));
    assert!(stderr(&o).contains("no numbers"));
}

#[test]
fn help_and_version() {
    let o = spark(&["--help"], None);
    assert!(o.status.success());
    assert!(stdout(&o).starts_with("usage: spark"));
    let o = spark(&["--version"], None);
    assert_eq!(stdout(&o), format!("spark {}\n", env!("CARGO_PKG_VERSION")));
}
