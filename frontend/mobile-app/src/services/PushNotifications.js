import PushNotification from 'react-native-push-notification';

PushNotification.configure({
  onRegister: function(token) {
    console.log("PushNotification Token:", token);
  },
  onNotification: function(notification) {
    console.log("Notification:", notification);
  },
  onAction: function(notification) {
    console.log("Action:", notification.action);
  },
  onRegistrationError: function(err) {
    console.error("PushNotification Registration Error:", err);
  },
  permissions: {
    alert: true,
    badge: true,
    sound: true,
  },
  popInitialNotification: true,
  requestPermissions: true,
});
