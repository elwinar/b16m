use reqwest;
use serde;
use std::collections;
use std::env;
use std::fs;
use std::io::Read;

macro_rules! fatal {
    ($($tt:tt)*) => {{
        use std::io::Write;
        writeln!(&mut ::std::io::stderr(), $($tt)*).unwrap();
        ::std::process::exit(1)
    }}
}

#[derive(Debug, Default, serde::Deserialize)]
#[serde(default)]
struct Config {
    scheme: String,
    scheme_repository_url: String,
    #[serde(default = "default_scheme_list_url")]
    schemes_list_url: String,
    #[serde(default = "default_templates_list_url")]
    templates_list_url: String,
    applications: collections::HashMap<String, Application>,
}

#[derive(Debug, Default, serde::Deserialize)]
#[serde(default)]
struct Application {
    hook: String,
    template_repository_url: String,
    files: collections::HashMap<String, File>,
}

#[derive(Debug, Default, serde::Deserialize)]
#[serde(default)]
struct File {
    destination: String,
    mode: String,
    start_marker: String,
    end_marker: String,
}

fn main() {
    let config = match fs::File::open("/home/elwinar/.config/b16m/config.yaml") {
        Ok(res) => res,
        Err(err) => fatal!("opening configuration file: {}", err),
    };
    let mut config: Config = match serde_yaml::from_reader(&config) {
        Ok(res) => res,
        Err(err) => fatal!("parsing configuration file: {}", err),
    };

    let args: Vec<String> = env::args().collect();
    match args.len() {
        1 => {}
        2 => {
            config.scheme = args[1].clone();
        }
        3 => {
            config.scheme = args[1].clone();
            config.scheme_repository_url = args[2].clone();
        }
        _ => fatal!("too many arguments"),
    };

    let client = reqwest::blocking::Client::new();

    let mut res = match client.get(&config.schemes_list_url).send() {
        Ok(res) => res,
        Err(err) => fatal!("retrieving schemes list: {}", err),
    };

    let mut body = String::new();
    if let Err(err) = res.read_to_string(&mut body) {
        fatal!("reading response body: {}", err);
    }

    if !res.status().is_success() {
        fatal!("unexpected status: {} {}", res.status(), body);
    }

    let schemes_list: collections::HashMap<String, String> = match serde_yaml::from_str(&body) {
        Ok(res) => res,
        Err(err) => fatal!("parsing schemes list: {}", err),
    };

    println!("{:?}", schemes_list);
}

fn default_scheme_list_url() -> String {
    github_file_url("chriskempson", "base16-schemes-source", "list.yaml")
}

fn default_templates_list_url() -> String {
    github_file_url("chriskempson", "base16-templates-source", "list.yaml")
}

fn github_file_url(user: &str, repository: &str, url: &str) -> String {
    format!(
        "https://raw.githubusercontent.com/{}/{}/master/{}",
        user, repository, url
    )
}
