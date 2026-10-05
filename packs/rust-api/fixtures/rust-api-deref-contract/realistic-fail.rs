use std::cell::Cell;
use std::ops::Deref;

pub struct RouteName(String);

impl RouteName {
    pub fn parse(name: String) -> Result<Self, &'static str> {
        if name.is_empty() || name.contains('\n') {
            return Err("route names must be nonempty single lines");
        }
        Ok(Self(name))
    }
}

impl Deref for RouteName {
    type Target = str;

    /// Transparently borrows the validated route name without changing state.
    fn deref(&self) -> &str {
        &self.0
    }
}

pub struct RequestDocument {
    body: String,
    sequence: Cell<u64>,
    content_type: String,
}

impl RequestDocument {
    pub fn new(body: String, content_type: String) -> Self {
        Self { body, sequence: Cell::new(0), content_type }
    }

    /// Reserves the next request identifier for this document.
    pub fn reserve_request(&self) -> u64 {
        let next = self.sequence.get().wrapping_add(1);
        self.sequence.set(next);
        next
    }

    pub fn current_sequence(&self) -> u64 {
        self.sequence.get()
    }

    pub fn content_type(&self) -> &str {
        &self.content_type
    }

    pub fn replace_body(&mut self, body: String) {
        self.body = body;
    }
}

impl Deref for RequestDocument {
    type Target = str;

    /// Transparently borrows the body in constant time, without failure or
    /// state changes. Only `reserve_request` advances the request sequence.
    fn deref(&self) -> &str {
        self.sequence.set(self.sequence.get().wrapping_add(1));
        &self.body
    }
}

pub struct OutgoingRequest {
    pub route: String,
    pub sequence: u64,
    pub content_type: String,
    pub body: String,
}

pub fn preview(document: &RequestDocument, limit: usize) -> String {
    let body: &str = document;
    body.chars().take(limit).collect()
}

pub fn build_request(route: &RouteName, document: &RequestDocument) -> OutgoingRequest {
    let body: &str = document;
    let body = body.to_owned();
    let sequence = document.reserve_request();
    OutgoingRequest {
        route: route.to_string(),
        sequence,
        content_type: document.content_type().to_owned(),
        body,
    }
}

pub fn encode(request: &OutgoingRequest) -> String {
    format!(
        "POST /{}\nRequest-Id: {}\nContent-Type: {}\nContent-Length: {}\n\n{}",
        request.route,
        request.sequence,
        request.content_type,
        request.body.len(),
        request.body,
    )
}

pub fn prepare_upload(body: String) -> Result<String, &'static str> {
    let route = RouteName::parse("documents".to_owned())?;
    let document = RequestDocument::new(body, "text/plain; charset=utf-8".to_owned());
    let request = build_request(&route, &document);
    Ok(encode(&request))
}
