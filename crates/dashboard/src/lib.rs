pub mod api;
pub mod components;
pub mod pages;

use leptos::prelude::*;
use leptos_meta::*;
use pages::DashboardPage;

#[component]
pub fn App() -> impl IntoView {
    provide_meta_context();

    view! {
        <Meta charset="utf-8" />
        <Meta name="viewport" content="width=device-width, initial-scale=1" />
        <Title text="button-hub Dashboard"/>
        <DashboardPage />
    }
}
