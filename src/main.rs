use serde;
use std::collections;
use std::fs;

#[derive(Debug, Default, serde::Deserialize)]
#[serde(default)]
struct Config {
    scheme: String,
    scheme_repository_url: String,
    schemes_list_url: String,
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
    let config: Config = match serde_yaml::from_reader(&config) {
        Ok(res) => res,
        Err(err) => panic!("parsing configuration file: {:?}", err),
    };

    println!("config: {:?}", config);
}
