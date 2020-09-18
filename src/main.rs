use serde::Deserialize;
use std::fs::File;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct Config {
    scheme: String,
    scheme_repository_url: String,
    schemes_list_url: String,
    templates_list_url: String,
}

fn main() {
    let config = match File::open("/home/elwinar/.config/b16m/config.yaml") {
        Ok(res) => res,
        Err(err) => panic!("opening configuration file: {:?}", err),
    };
    let config: Config = match serde_yaml::from_reader(&config) {
        Ok(res) => res,
        Err(err) => panic!("parsing configuration file: {:?}", err),
    };

    println!("config: {:?}", config);
}
