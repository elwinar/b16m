use reqwest;
use serde;
use std::collections;
use std::env;
use std::error;
use std::fs;

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

    let schemes_list: collections::HashMap<String, String> =
        match get_yaml_file(client, config.schemes_list_url) {
            Ok(v) => v,
            Err(e) => fatal!("retrieving schemes list: {}", e),
        };

    let templates_list: collections::HashMap<String, String> =
        match get_yaml_file(client, config.templates_list_url) {
            Ok(v) => v,
            Err(e) => fatal!("retrieving templates list: {}", e),
        };
}

fn get_yaml_file<T: for<'a> serde::Deserialize<'a>>(
    client: reqwest::blocking::Client,
    url: String,
) -> Result<T, Box<dyn error::Error>> {
    let body = get_file(client, url)?;
    Ok(serde_yaml::from_str::<T>(&body)?)
}

fn get_file(client: reqwest::blocking::Client, url: String) -> Result<String, reqwest::Error> {
    Ok(client.get(&url).send()?.error_for_status()?.text()?)
}
