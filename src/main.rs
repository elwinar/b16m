use reqwest;
use serde;
use std::collections;
use std::env;
use std::fs;
use std::io::Read;

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
        Err(err) => panic!("opening configuration file: {:?}", err),
    };
    let mut config: Config = match serde_yaml::from_reader(&config) {
        Ok(res) => res,
        Err(err) => panic!("parsing configuration file: {:?}", err),
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
        _ => panic!("too many arguments"),
    };

    let client = reqwest::blocking::Client::new();

    let mut res = match client.get(&config.schemes_list_url).send() {
        Ok(res) => res,
        Err(err) => panic!("retrieving schemes list: {:?}", err),
    };

    let mut body = String::new();
    match res.read_to_string(&mut body) {
        Err(err) => panic!("reading response body: {:?}", err),
        _ => {}
    }

    if !res.status().is_success() {
        panic!("unexpected status: {:?} {}", res.status(), body);
    }

    println!("{}", body);
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
