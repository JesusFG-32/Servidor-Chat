CREATE USER IF NOT EXISTS 'chat_user'@'localhost' IDENTIFIED BY 'chat_pass';
GRANT ALL PRIVILEGES ON chat_app.* TO 'chat_user'@'localhost';
FLUSH PRIVILEGES;
