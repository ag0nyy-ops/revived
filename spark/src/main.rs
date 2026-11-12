use spark::cli::{self, Env, Options};
use spark::render::Charset;
use std::io::{self, IsTerminal, Read, Write};
use std::process::ExitCode;

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let opts = match Options::parse(&args) {
        Ok(o) => o,
        Err(e) => {
            eprintln!("spark: {e}");
            return ExitCode::from(2);
        }
    };
    if opts.help {
        println!("{}", cli::USAGE);
        return ExitCode::SUCCESS;
    }
    if opts.version {
        println!("spark {}", env!("CARGO_PKG_VERSION"));
        return ExitCode::SUCCESS;
    }

    let input = if !opts.values.is_empty() {
        opts.values.join(" ")
    } else if io::stdin().is_terminal() {
        // Nothing piped in and nothing on the command line: show help
        // instead of sitting there waiting for input.
        eprintln!("{}", cli::USAGE);
        return ExitCode::from(2);
    } else {
        let mut buf = String::new();
        if let Err(e) = io::stdin().read_to_string(&mut buf) {
            eprintln!("spark: reading stdin: {e}");
            return ExitCode::FAILURE;
        }
        buf
    };

    let env = Env {
        charset: Charset::from_locale(|k| std::env::var(k).ok()),
        stdout_is_tty: io::stdout().is_terminal(),
        no_color: std::env::var_os("NO_COLOR").is_some_and(|v| !v.is_empty()),
    };
    match cli::render(&opts, &input, &env) {
        Ok(lines) => {
            let mut out = io::stdout().lock();
            for line in lines {
                // A closed pipe (spark ... | head) isn't worth a panic.
                if writeln!(out, "{line}").is_err() {
                    break;
                }
            }
            ExitCode::SUCCESS
        }
        Err(e) => {
            eprintln!("spark: {e}");
            ExitCode::FAILURE
        }
    }
}
