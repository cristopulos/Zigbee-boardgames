use crate::event::Event;
use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;

pub type HandlerFn = Arc<dyn Fn(Event) -> Pin<Box<dyn Future<Output = ()> + Send>> + Send + Sync>;

pub struct Button {
    pub id: String,
    pub handler: HandlerFn,
}

impl Button {
    pub fn new<F, Fut>(id: impl Into<String>, handler: F) -> Self
    where
        F: Fn(Event) -> Fut + Send + Sync + 'static,
        Fut: Future<Output = ()> + Send + 'static,
    {
        Self {
            id: id.into(),
            handler: Arc::new(move |e| Box::pin(handler(e))),
        }
    }
}
