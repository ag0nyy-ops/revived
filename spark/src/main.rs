use std::process::ExitCode;

const USAGE: &str = "usage: spark [options] [--] [numbers...]

Prints a sparkline for the given numbers, or for numbers read from stdin.

options:
  -h, --help     show this help
  -V, --version  show the version";

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args.iter().any(|a| a == "-h" || a == "--help") {
        println!("{USAGE}");
        return ExitCode::SUCCESS;
    }
    if args.iter().any(|a| a == "-V" || a == "--version") {
        println!("spark {}", env!("CARGO_PKG_VERSION"));
        return ExitCode::SUCCESS;
    }
    eprintln!("{USAGE}");
    ExitCode::from(2)
}
