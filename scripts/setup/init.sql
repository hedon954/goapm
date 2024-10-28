CREATE DATABASE IF NOT EXISTS goapm;

CREATE TABLE IF NOT EXISTS `goapm`.`t_user` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `uid` VARCHAR(64) NOT NULL UNIQUE,
    `name` VARCHAR(64) NOT NULL,
    `age` INT NOT NULL,
    `gender` VARCHAR(16) NOT NULL,
    `address` VARCHAR(128) NOT NULL,
    `phone` VARCHAR(16) NOT NULL,
    `email` VARCHAR(64) NOT NULL,
    `salary` DECIMAL(10, 2) NOT NULL,
    `ctime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `utime` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

## insert if not exists
INSERT INTO `goapm`.`t_user` (`uid`, `name`, `age`, `gender`, `address`, `phone`, `email`, `salary`) VALUES
('u001', 'Alice', 25, 'Female', '123 Main St, Anytown, USA', '123-456-7890', 'alice@example.com', 50000.00),
('u002', 'Bob', 30, 'Male', '456 Elm St, Anytown, USA', '098-765-4321', 'bob@example.com', 60000.00)
ON DUPLICATE KEY UPDATE `uid` = `uid`;
