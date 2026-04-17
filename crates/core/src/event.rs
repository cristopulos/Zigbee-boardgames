use chrono::{DateTime, Utc};
use serde::{de::Visitor, Deserialize, Serialize, Serializer};

#[derive(Debug, Clone, PartialEq)]
pub enum ActionType {
    Single,
    Double,
    LongPress,
    Unknown(String),
}

impl Serialize for ActionType {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        let s = match self {
            ActionType::Single => "Single",
            ActionType::Double => "Double",
            ActionType::LongPress => "LongPress",
            ActionType::Unknown(inner) => {
                return serializer.serialize_str(&format!("Unknown({})", inner))
            }
        };
        serializer.serialize_str(s)
    }
}

impl<'de> Deserialize<'de> for ActionType {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ActionTypeVisitor;

        impl<'de> Visitor<'de> for ActionTypeVisitor {
            type Value = ActionType;

            fn expecting(&self, formatter: &mut std::fmt::Formatter) -> std::fmt::Result {
                formatter.write_str("a string representing an action type")
            }

            fn visit_str<E>(self, value: &str) -> Result<ActionType, E>
            where
                E: serde::de::Error,
            {
                Ok(match value {
                    "Single" => ActionType::Single,
                    "Double" => ActionType::Double,
                    "LongPress" => ActionType::LongPress,
                    s if s.starts_with("Unknown(") && s.ends_with(")") => {
                        let inner = &s[8..s.len() - 1];
                        ActionType::Unknown(inner.to_string())
                    }
                    _ => ActionType::Unknown(value.to_string()),
                })
            }
        }

        deserializer.deserialize_str(ActionTypeVisitor)
    }
}

impl From<&str> for ActionType {
    fn from(s: &str) -> Self {
        match s {
            "single" => ActionType::Single,
            "double" => ActionType::Double,
            "long_press" | "hold" | "long" => ActionType::LongPress,
            _ => ActionType::Unknown(s.to_string()),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Event {
    pub button_id: String,
    pub action: ActionType,
    pub battery: Option<u8>,
    pub timestamp: DateTime<Utc>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn action_type_from_single() {
        assert_eq!(ActionType::from("single"), ActionType::Single);
    }

    #[test]
    fn action_type_from_double() {
        assert_eq!(ActionType::from("double"), ActionType::Double);
    }

    #[test]
    fn action_type_from_long_press() {
        assert_eq!(ActionType::from("long_press"), ActionType::LongPress);
    }

    #[test]
    fn action_type_from_hold() {
        assert_eq!(ActionType::from("hold"), ActionType::LongPress);
    }

    #[test]
    fn action_type_from_long() {
        assert_eq!(ActionType::from("long"), ActionType::LongPress);
    }

    #[test]
    fn action_type_from_bogus() {
        assert_eq!(
            ActionType::from("bogus"),
            ActionType::Unknown("bogus".to_string())
        );
    }

    #[test]
    fn serialize_known_variants() {
        assert_eq!(json_str(&ActionType::Single), "\"Single\"");
        assert_eq!(json_str(&ActionType::Double), "\"Double\"");
        assert_eq!(json_str(&ActionType::LongPress), "\"LongPress\"");
    }

    #[test]
    fn serialize_unknown() {
        assert_eq!(
            json_str(&ActionType::Unknown("long".to_string())),
            "\"Unknown(long)\""
        );
    }

    #[test]
    fn deserialize_known_variants() {
        assert_eq!(parse_action("\"Single\""), ActionType::Single);
        assert_eq!(parse_action("\"Double\""), ActionType::Double);
        assert_eq!(parse_action("\"LongPress\""), ActionType::LongPress);
    }

    #[test]
    fn deserialize_unknown_roundtrip() {
        let original = ActionType::Unknown("shake".to_string());
        let json = json_str(&original);
        assert_eq!(parse_action(&json), original);
    }

    #[test]
    fn deserialize_plain_unknown() {
        assert_eq!(
            parse_action("\"weird_action\""),
            ActionType::Unknown("weird_action".to_string())
        );
    }

    #[test]
    fn event_json_roundtrip() {
        let event = Event {
            button_id: "btn1".to_string(),
            action: ActionType::LongPress,
            battery: Some(85),
            timestamp: chrono::Utc::now(),
        };
        let json = serde_json::to_string(&event).unwrap();
        let parsed: Event = serde_json::from_str(&json).unwrap();
        assert_eq!(event.action, parsed.action);
        assert_eq!(event.button_id, parsed.button_id);
        assert_eq!(event.battery, parsed.battery);
    }

    fn json_str(action: &ActionType) -> String {
        serde_json::to_string(action).unwrap()
    }

    fn parse_action(json: &str) -> ActionType {
        serde_json::from_str(json).unwrap()
    }
}
