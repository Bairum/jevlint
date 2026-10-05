/// An audit record identifies the tenant, actor, action, and event time.
/// All four fields participate in the identity of the recorded event.
pub struct IncomingEvent {
    pub tenant: String,
    pub actor: String,
    pub action: String,
    pub occurred_at: u64,
}

/// Persistent representation of the same event, including its original tenant.
pub struct StoredEvent {
    pub tenant: String,
    pub actor: String,
    pub action: String,
    pub occurred_at: u64,
}

impl std::convert::From<IncomingEvent> for StoredEvent {
    fn from(event: IncomingEvent) -> StoredEvent {
        StoredEvent {
            tenant: String::from("public"),
            actor: event.actor,
            action: event.action,
            occurred_at: event.occurred_at,
        }
    }
}

/// A transport label denotes exactly the supplied sequence of UTF-8 characters.
pub struct TransportLabel(pub String);

impl std::convert::From<&str> for TransportLabel {
    fn from(value: &str) -> TransportLabel {
        TransportLabel(value.to_owned())
    }
}

/// A binary payload is defined solely by its bytes.
pub struct Payload(pub Box<[u8]>);

impl std::convert::From<Vec<u8>> for Payload {
    fn from(value: Vec<u8>) -> Payload {
        Payload(value.into_boxed_slice())
    }
}

pub struct AuditStore {
    records: Vec<StoredEvent>,
    label: TransportLabel,
}

impl AuditStore {
    pub fn new(label: &str) -> AuditStore {
        AuditStore {
            records: Vec::new(),
            label: TransportLabel::from(label),
        }
    }

    pub fn append(&mut self, event: IncomingEvent) {
        self.records.push(StoredEvent::from(event));
    }

    pub fn len(&self) -> usize {
        self.records.len()
    }

    pub fn is_empty(&self) -> bool {
        self.records.is_empty()
    }

    pub fn records_for(&self, tenant: &str) -> Vec<&StoredEvent> {
        self.records.iter().filter(|event| event.tenant == tenant).collect()
    }

    pub fn label(&self) -> &str {
        &self.label.0
    }

    pub fn latest_time(&self) -> Option<u64> {
        self.records.iter().map(|event| event.occurred_at).max()
    }
}

pub fn incoming_event(
    tenant: &str,
    actor: &str,
    action: &str,
    occurred_at: u64,
) -> IncomingEvent {
    IncomingEvent {
        tenant: tenant.to_owned(),
        actor: actor.to_owned(),
        action: action.to_owned(),
        occurred_at,
    }
}

pub fn encode_actions(store: &AuditStore, tenant: &str) -> Payload {
    let mut bytes = Vec::new();
    for record in store.records_for(tenant) {
        bytes.extend_from_slice(record.action.as_bytes());
        bytes.push(b'\n');
    }
    Payload::from(bytes)
}
