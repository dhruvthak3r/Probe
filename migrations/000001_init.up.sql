
CREATE TABLE `monitor` (
  `monitor_id` bigint NOT NULL AUTO_INCREMENT,
  `monitor_name` varchar(255) NOT NULL,
  `url` varchar(2048) NOT NULL,
  `frequency_seconds` int NOT NULL,
  `last_run_at` datetime DEFAULT NULL,
  `next_run_at` datetime DEFAULT NULL,
  `response_format` enum('string','json') NOT NULL,
  `http_method` enum('GET','POST','PUT','DELETE','PATCH','HEAD','OPTIONS') NOT NULL DEFAULT 'GET',
  `client_certificate` text,
  `is_active` tinyint(1) DEFAULT '1',
  `is_mock` tinyint(1) DEFAULT '0',
  `status` enum('idle','running') DEFAULT 'idle',
  `connection_timeout` bigint DEFAULT NULL,
  `request_body` text,
  PRIMARY KEY (`monitor_id`)
) ENGINE=InnoDB AUTO_INCREMENT=23181 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `monitor_accepted_status_codes` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `status_code` int DEFAULT NULL,
  `monitor_id` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `monitor_id` (`monitor_id`),
  CONSTRAINT `monitor_accepted_status_codes_ibfk_1` FOREIGN KEY (`monitor_id`) REFERENCES `monitor` (`monitor_id`)
) ENGINE=InnoDB AUTO_INCREMENT=61894 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


CREATE TABLE `monitor_request_headers` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(256) NOT NULL,
  `value` text NOT NULL,
  `monitor_id` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `monitor_id` (`monitor_id`),
  CONSTRAINT `monitor_request_headers_ibfk_1` FOREIGN KEY (`monitor_id`) REFERENCES `monitor` (`monitor_id`)
) ENGINE=InnoDB AUTO_INCREMENT=61888 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;



CREATE TABLE `monitor_response_headers` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(256) NOT NULL,
  `value` text NOT NULL,
  `monitor_id` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `monitor_id` (`monitor_id`),
  CONSTRAINT `monitor_response_headers_ibfk_1` FOREIGN KEY (`monitor_id`) REFERENCES `monitor` (`monitor_id`)
) ENGINE=InnoDB AUTO_INCREMENT=41259 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;



CREATE TABLE `results` (
  `result_id` bigint NOT NULL AUTO_INCREMENT,
  `monitor_id` bigint DEFAULT NULL,
  `loc_id` int DEFAULT NULL,
  `status` enum('DOWN','UP') DEFAULT NULL,
  `status_code` int DEFAULT NULL,
  `dns_response_time` bigint DEFAULT NULL,
  `connection_time` bigint DEFAULT NULL,
  `tls_handshake_time` bigint DEFAULT NULL,
  `reason` text,
  `checked_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `first_byte_time` bigint DEFAULT NULL,
  `download_time` bigint DEFAULT NULL,
  `throughput` double DEFAULT NULL,
  `resolved_ip` varchar(36) DEFAULT NULL,
  `response_time` bigint DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`result_id`),
  KEY `monitor_id` (`monitor_id`),
  KEY `loc_id` (`loc_id`),
  CONSTRAINT `results_ibfk_1` FOREIGN KEY (`monitor_id`) REFERENCES `monitor` (`monitor_id`)
) ENGINE=InnoDB AUTO_INCREMENT=254645 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
