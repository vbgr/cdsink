use std::collections::HashMap;

use apache_avro::AvroSchema;
use fake::Dummy;
use fake::faker::address::en::*;
use fake::{Fake, Faker};
use serde::Serialize;

#[derive(Serialize, AvroSchema, Debug, Clone, Dummy)]
pub struct Contact {
    #[dummy(faker = "fake::faker::internet::en::SafeEmail()")]
    pub email: String,
    #[dummy(faker = "fake::faker::phone_number::en::PhoneNumber()")]
    pub phone: Option<String>,
}

#[allow(dead_code)]
impl Contact {
    pub fn new() -> Self {
        Faker.fake()
    }
}

#[derive(Serialize, AvroSchema, Debug, Clone, Dummy)]
pub struct Address {
    #[dummy(faker = "CountryName()")]
    pub country: String,
    #[dummy(faker = "CityName()")]
    pub city: String,
    #[dummy(faker = "StreetName()")]
    pub street: String,
    #[dummy(faker = "ZipCode()")]
    pub zipcode: Option<String>,
}

#[allow(dead_code)]
impl Address {
    pub fn new() -> Self {
        Faker.fake()
    }
}

#[derive(Serialize, AvroSchema, Debug, Clone, Dummy)]
pub struct Person {
    #[dummy(faker = "fake::faker::barcode::en::Isbn()")]
    pub id: String,
    #[dummy(faker = "fake::faker::name::en::FirstName()")]
    pub first_name: String,
    #[dummy(faker = "fake::faker::name::en::LastName()")]
    pub last_name: String,
    #[dummy(faker = "18..80")]
    pub age: Option<i32>,
    pub contacts: HashMap<String, Contact>,
    pub addresses: Option<Vec<Address>>,
}

#[allow(dead_code)]
impl Person {
    pub fn new() -> Self {
        Faker.fake()
    }
}
