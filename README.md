# Real-Time Data Streaming Pipeline with Kafka, Docker, and Go

This project implements a real-time data streaming pipeline using **Apache Kafka**, **Docker**, and **Go**. It involves creating a system to consume, process, and produce data messages in Kafka topics, while ensuring scalability, fault tolerance, and efficient message handling. The pipeline consists of the following components:

1. **Kafka** for message ingestion and distribution.
2. **Docker** for containerization of services.
3. **Go-based consumer service** to process and produce data.
4. **Python-based producer service** for simulating data generation and pushing it to Kafka.

## Overview of the Solution

The solution involves setting up a Kafka consumer in Go that consumes messages from a Kafka topic (`user-login`), processes them based on certain rules, and publishes the processed data to another Kafka topic (`processed-user-login`). Any invalid messages are sent to a Dead Letter Queue (DLQ) topic (`user-login-dlq`). 

### Key Features:
- **Kafka consumer**: Reads messages from a Kafka topic (`user-login`), processes them based on certain checks, and publishes valid messages to the `processed-user-login` topic.
- **Dead Letter Queue (DLQ)**: Invalid messages are sent to the `user-login-dlq` topic.
- **Fault tolerance and retries**: The consumer ensures that messages are processed even in case of temporary issues, using retry logic with exponential backoff.
- **Graceful shutdown**: The application handles shutdown signals to close Kafka consumer and producer connections cleanly.

## Design Choices and Considerations

### 1. **Kafka Topics and Data Flow**
   - **Input Topic**: `user-login` – This is the main topic where messages are consumed from. Messages in this topic are expected to contain user login information in JSON format.
   - **Output Topic**: `processed-user-login` – After processing, valid messages are published to this topic.
   - **Dead Letter Queue (DLQ)**: `user-login-dlq` – Any invalid messages (e.g., missing fields or invalid data) are sent to this DLQ for further inspection.

### 2. **Consumer Logic in Go**
   - The consumer subscribes to the `user-login` topic.
   - It processes each message and validates fields like `UserID`, `AppVersion`, and `DeviceType`. If a message fails validation, it is sent to the DLQ.
   - The consumer uses a worker pool model to handle message processing concurrently, improving throughput.
   - Kafka consumer offsets are managed manually to provide greater control over message acknowledgment, which ensures that messages are not lost in case of a failure.

### 3. **Choice of Kafka, third party libraries and implementaton language** 
   - **Design Priorities**: The design is built with efficiency, scalability, and fault tolerance in mind. 
   - **Go as the programming language**: Go was chosen for its lightweight, asynchronous, and efficient nature. It is well-suited for building scalable and performant systems that interact with Kafka, where message consumption and processing can occur concurrently without blocking other operations. Go's simplicity, fast compilation time, and robust concurrency model (goroutines and channels) make it an ideal choice for this type of real-time data pipeline.
   - **`confluentinc/confluent-kafka-go`**: This library was selected for its minimal configuration, high performance, and strong integration with the Confluent Kafka ecosystem. It provides a reliable, efficient, and straightforward interface for Kafka consumers and producers, making it the most practical choice for this project. Alternatives like `goka`, `kafka-go`, and `Shopify/sarama` could also be used, but `confluent-kafka-go` was chosen due to its direct support for Kafka's native protocol and integration with Kafka.
   - **Kafka as the backbone of the data pipeline**: Kafka is used for handling high throughput of streaming data. Kafka's partitioning model and fault tolerance through replication provide scalability and reliability, ensuring the system can handle large volumes of data without significant performance degradation.

### 4. **Fault Tolerance and Scalability**
   - **Retries**: The consumer implements exponential backoff for retries to avoid overloading the Kafka brokers in case of transient issues.
   - **Concurrency**: A worker pool is used to process multiple messages concurrently, improving throughput and scalability.
   - **Graceful Shutdown**: The consumer listens for termination signals (e.g., SIGTERM) and shuts down Kafka connections cleanly, ensuring no data is lost.

## Running the Project Locally

### Prerequisites
- **Docker** and **Docker Compose** must be installed. 
- **Kafka** and **Zookeeper** will be run as Docker containers.

### Steps to Run the Project

1. Clone this repository:
   ```bash
   git clone git@github.com:dmitryro/kafka-pipeline.git > data_pipeline 
   cd data_pipeline
   ```

2. Build and start the services using Docker Compose:
   ```bash
   docker-compose up --build
   ```

   This command will build all the Docker images and start the following services:
   - **Kafka**: A Kafka broker running on port `9092` (internal) and `29092` (external).
   - **Zookeeper**: A Zookeeper instance used by Kafka for coordination.
   - **Producer Service (Python)**: A producer that generates and sends data to Kafka.
   - **Consumer Service (Go)**: The consumer that processes and publishes data to Kafka topics.

3. After running the above command, the services should be up and running. You can verify this by checking the logs of the consumer:
   ```bash
   docker logs pipeline-consumer
   ```

   You can also use Kafka's `kafka-console-consumer` tool to check the messages in the `processed-user-login` topic:
   ```bash
   kafka-console-consumer --bootstrap-server localhost:29092 --topic processed-user-login --from-beginning
   ```

### Environment Variables

- **LEVEL**: Controls the logging level. Set to `DEBUG` in `.env` for development and `INFO` for production.
- **KAFKA_LISTENER**: The Kafka broker URL for internal communication (e.g., `kafka:9092`).
- **KAFKA_BROKER_URL**: The Kafka broker URL for external communication (e.g., `localhost:29092`).
- **KAFKA_CREATE_TOPICS**: Comma-separated list of Kafka topics to create on startup.
- **KAFKA_ZOOKEEPER_CONNECT**: Connection string for Zookeeper.

See `.env` for all available environment variables and their descriptions.

### Sample .env File

```env
LEVEL=DEBUG
PROJECT_NAME=pipeline
KAFKA_LISTENER=kafka://kafka:9092
KAFKA_BROKER_URL=kafka:9092
KAFKA_LISTENERS=LISTENER_INTERNAL://kafka:9092,LISTENER_EXTERNAL://localhost:29092
KAFKA_ADVERTISED_LISTENERS=LISTENER_INTERNAL://kafka:9092,LISTENER_EXTERNAL://localhost:29092
KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=LISTENER_INTERNAL:PLAINTEXT,LISTENER_EXTERNAL:PLAINTEXT
KAFKA_INTER_BROKER_LISTENER_NAME=LISTENER_INTERNAL
KAFKA_ADVERTISED_HOST_NAME=localhost
KAFKA_BROKER_ID=1
KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181
KAFKA_CREATE_TOPICS="user-login:1:1,processed-user-login:1:1,user-login-dlq:1:1"
ZOOKEEPER_CLIENT_PORT=2181
ZOOKEEPER_TICK_TIME=2000
ZOO_MY_ID=1
ZOO_PORT=2181
ZOO_SERVERS="server.1=zookeeper:2888:3888"
CONSUMER_GROUP=user-group
BOOTSTRAP_SERVERS=kafka:9092
ENABLE_AUTO_COMMIT=false
SOCKET_TIMEOUT=30000
SESSION_TIMEOUT=30000
AUTO_OFFSET_RESET=earliest
INPUT_TOPIC=user-login
OUTPUT_TOPIC=processed-user-login
DLQ_TOPIC=user-login-dlq
```

## Production Readiness

### 1. **Deployment to Kubernetes**
   - The solution can be deployed to **Kubernetes** for managing and scaling services in production. Kubernetes helps in automating the deployment, scaling, and management of containerized applications.
   - **Helm** can be used for easy configuration management and deployment of the system. Helm charts simplify the deployment process by packaging Kubernetes resources like deployments, services, and persistent volumes into reusable templates.
   - **Deployment on AWS EKS (Elastic Kubernetes Service)** or other managed Kubernetes services is recommended for better scalability and ease of maintenance. EKS provides a managed Kubernetes environment that can be scaled as needed, with built-in security, monitoring, and high availability.
   - The system should include horizontal scaling for both the Kafka producer and consumer services, ensuring that the pipeline can handle a growing volume of data without downtime.

### 2. **Monitoring and Logging in Production**
   - For monitoring, integrate with **Prometheus** and **Grafana** to track the health of Kafka, consumer, and producer services.
   - **Prometheus** will gather metrics, while **Grafana** can be used to create dashboards for real-time monitoring.
   - Use the **ELK stack (Elasticsearch, Logstash, Kibana)** for centralized logging. Logs from all services, including Kafka brokers, producers, and consumers, can be aggregated in Elasticsearch, and visualized in Kibana for troubleshooting and performance monitoring.

### 3. **Kafka in Production**
   - **Replication**: Kafka topics should have a replication factor greater than 1 for high availability. This ensures that Kafka data remains available even if a broker fails.
   - **Partitioning**: Kafka topic partitioning should be configured according to throughput requirements. More partitions allow better distribution of the data across multiple Kafka brokers, improving scalability.
   - **Kafka Connect** can be used for integrating external systems (such as databases or third-party APIs) to produce or consume data from Kafka topics.

### 4. **Scaling Considerations**
   - The Kafka cluster should be scaled horizontally by adding more brokers to the Kafka cluster as needed.
   - The consumer application should also be scaled horizontally by adding more pods or containers. Each consumer should be part of a consumer group to ensure that messages are processed in parallel across multiple instances.

### 5. **Automated Deployments and CI/CD**
   - Implement a **CI/CD pipeline** using **GitLab CI**, **Jenkins**, or **GitHub Actions** to automate the testing, building, and deployment of the system to Kubernetes.
   - The pipeline should include steps to:
     - Build Docker images for the producer and consumer services.
     - Push the Docker images to a container registry (e.g., Docker Hub, Amazon ECR).
     - Deploy the services to Kubernetes (e.g., using Helm charts).
     - Monitor health and automatically scale services based on resource utilization or incoming data volume.

### 6. **Security and Compliance**
   - Implement **role-based access control (RBAC)** in Kubernetes to ensure that only authorized users and services can access Kafka topics or deploy updates to the pipeline.
   - **Audit logging** for all Kafka interactions can help with security and compliance, especially in regulated industries.
   - **TLS encryption** :Ensure secure communication with Kafka brokers by enabling **SSL/TLS** encryption for both producers and consumers.
   - Use **IAM roles** for secure access to cloud services like S3 or MSK, ensuring least privilege access.
   #### **IAM Roles for Kafka**

   Cloud-based Kafka services like Amazon MSK (Managed Streaming for Apache Kafka) and Confluent Cloud rely on Identity and Access Management (IAM) roles for securing access to Kafka resources. IAM roles are used to authenticate and authorize clients, services, and applications interacting with Kafka clusters.

   ##### **Amazon MSK IAM Roles**

   In MSK, IAM roles are used to control access to your Kafka brokers and Kafka data within AWS. You can use IAM roles to:

   - **Grant permissions to clients:** Through the use of IAM policies, you can control which users, roles, or services can produce, consume, or administer Kafka topics.
   - **Authenticate clients:** MSK supports **IAM authentication** for producers and consumers to securely connect to Kafka brokers. IAM roles can be assigned to EC2 instances or services like AWS Lambda to authenticate without using traditional usernames and passwords.
   - **Access Control:** Policies can be attached to IAM roles, controlling access based on Kafka resources like topics and consumer groups.

   IAM roles for MSK are managed through AWS Identity and Access Management (IAM), and the appropriate permissions must be granted to allow the Kafka client applications to interact with MSK clusters.

   ##### Example: IAM Policy for MSK Consumer

   ```json
   {
       "Version": "2012-10-17",
       "Statement": [
           {
               "Effect": "Allow",
               "Action": [
                   "kafka:DescribeCluster",
                   "kafka:DescribeTopic",
                   "kafka:ListTopics",
                   "kafka:GetRecords",
                   "kafka:Consume"
               ],
               "Resource": "arn:aws:kafka:region:account-id:cluster/cluster-name/*"
           }
       ]
   }
   ```

   #### Confluent Cloud IAM Roles

   Confluent Cloud provides a robust IAM (Identity and Access Management) system to control access to Kafka resources. It integrates with cloud-native IAM systems like AWS IAM, Google Cloud IAM, and Azure AD to enable seamless and secure access control. With Confluent Cloud, you can define fine-grained permissions for managing Kafka clusters, topics, consumer groups, and other resources.

   ##### **Key IAM Roles in Confluent Cloud**

   - **Administrator**: Full access to all resources and configurations within the Confluent Cloud environment. This role can manage Kafka clusters, create and delete topics, and manage IAM policies.
     
   - **Kafka Cluster Admin**: Can create and manage Kafka clusters, configure brokers, and manage topic configurations. However, they do not have access to non-Kafka services like connectors, schemas, or user management.
     
   - **Developer**: Can produce and consume messages to/from Kafka topics and create topics, but has limited access to administrative functionalities. Developers typically focus on managing their specific applications.
     
   - **Viewer**: Can only view the configuration of Kafka resources, including topic details, consumer groups, and cluster configurations. This role does not allow any changes or access to message data.
     
   - **Schema Registry Admin**: Can manage schemas within the Schema Registry but does not have access to Kafka cluster or other non-schema resources.

   ##### **Assigning IAM Roles in Confluent Cloud**

   IAM roles are assigned at different levels, including:

   - **Organization level**: Users can be assigned roles that give access to all resources within the Confluent Cloud organization.
   - **Cluster level**: Roles can be restricted to a specific Kafka cluster or specific topics within that cluster.
   - **Topic level**: Fine-grained access can be applied, such as allowing a user to only produce messages to a specific topic.

   Roles are assigned through the Confluent Cloud UI or via the API by the administrator.

   ##### **Best Practices for IAM Role Management in Confluent Cloud**

   - **Principle of Least Privilege**: Always assign the least amount of privilege necessary to perform the required tasks. For example, a developer should not be granted administrator permissions unless absolutely necessary.
   - **Use Role-based Access Control (RBAC)**: RBAC allows administrators to define roles with specific permissions for different users or services within the organization.
   - **Monitor Role Assignments**: Regularly review and audit IAM roles to ensure that only authorized users and services have access to sensitive Kafka resources.
   - **Use Multi-Factor Authentication (MFA)**: Enhance security by enabling MFA for users with elevated IAM roles, such as administrators.

### 7. **Fault Tolerance and High Availability**
   - Ensure that **Kafka brokers** are deployed in a fault-tolerant configuration with replication across multiple availability zones to avoid data loss in case of broker failure.
   - Kafka consumers and producers should be deployed in a manner that ensures high availability, possibly using multiple instances across different Kubernetes pods or nodes.
   - Implement **Health Checks** for Kafka brokers, producers, and consumers to monitor their availability and restart them automatically in case of failure.
   - Consider implementing **circuit breaker** patterns in case of failures in external systems.
   - Design the application to handle Kafka broker failures and allow for graceful recovery.

### 8. **Backup and Disaster Recovery**
   - **Kafka Backups**: Implement a backup strategy for Kafka logs and topic data. Periodic snapshots of the Kafka data can be taken to ensure recovery in case of catastrophic failure.
   - **Disaster Recovery Plan**: In the event of a disaster, ensure that backup data can be restored to a new Kafka cluster quickly, minimizing downtime and data loss.

### 9. **Cost Optimization**
   - Use **auto-scaling** in Kubernetes to adjust the number of producer and consumer pods based on workload, ensuring that the system can scale up during high data traffic and scale down during idle times.
   - Optimize the **Kafka cluster's storage** by adjusting the retention period of topics and using **log compaction** for certain topics to save disk space.
   - Monitor and adjust **instance types and resource allocation** for Kafka brokers and consumer services to avoid over-provisioning while ensuring adequate performance.

### 10. **Error Handling and Alerts**:
   - Implement comprehensive error handling to gracefully handle failures in Kafka message processing.
   - Set up alerts using tools like **Prometheus Alertmanager** or **Datadog** to monitor for issues like consumer lag, application crashes, and resource utilization.


## Scalability

As the dataset grows, the application should be designed to scale efficiently. Here are the key strategies for scaling:

1. **Horizontal Scaling of Consumers**:
   - You can scale the number of Kafka consumers to handle increased traffic. Kafka allows multiple consumers to read from the same topic by creating multiple consumer instances in different processes or containers. This ensures that the workload is distributed evenly.
   - Use a load balancer or Kubernetes to manage consumer scaling automatically based on CPU or memory usage.

2. **Kafka Partitioning**:
   - To improve throughput and distribute data processing more evenly, increase the number of partitions for Kafka topics. This allows consumers to read from different partitions in parallel, enhancing the throughput of the system.

3. **Backpressure Handling**:
   - In case of increased load, implement backpressure handling techniques, such as controlling the rate at which data is processed or batching the messages, to avoid overwhelming the system.

4. **Database Scaling**:
   - If the processed data is being stored in a database, ensure that the database can handle the increasing load. This may involve database sharding, read replicas, or using distributed databases that can scale horizontally.

5. **Cloud Resources**:
   - If using cloud services like AWS, GCP, or Azure, ensure auto-scaling is enabled for Kafka brokers and application instances. This ensures that the infrastructure adapts to growing loads without manual intervention.


## Troubleshooting Tips

If you encounter issues while running the project, here are some common problems and solutions:

### 1. **Kafka Consumer Not Receiving Messages**
   - **Cause**: The Kafka consumer may not be properly connected to the Kafka broker or may be misconfigured.
   - **Solution**:
     - Verify that the Kafka broker is running. You can check the logs of the Kafka container:
       ```bash
       docker logs pipeline-kafka
       ```
     - Ensure that the `KAFKA_BROKER_URL` environment variable is correctly set to the correct Kafka broker address in the `.env` file.
     - Check if the `user-login` topic exists. If not, create it using Kafka's CLI:
       ```bash
       kafka-topics.sh --create --topic user-login --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       ```

### 2. **Messages Going to Dead Letter Queue (DLQ)**
   - **Cause**: The consumer may be rejecting valid messages due to incorrect validation logic.
   - **Solution**:
     - Review the validation logic in the consumer code to ensure that the fields (e.g., `UserID`, `AppVersion`, `DeviceType`) are being validated correctly.
     - Check the logs for any errors related to the DLQ. You can use `docker logs` to inspect the consumer logs:
       ```bash
       docker logs pipeline-consumer
       ```

### 3. **Kafka Connection Timeout**
   - **Cause**: Kafka might not be reachable from your consumer or producer.
   - **Solution**:
     - Check if the Kafka and Zookeeper containers are running properly by inspecting their logs:
       ```bash
       docker logs pipeline-kafka
       docker logs pipeline-zookeeper
       ```
     - Ensure that the `KAFKA_LISTENER` and `KAFKA_BROKER_URL` environment variables in the `.env` file are correctly configured for internal and external communication.

### 4. **Producer Not Sending Messages to Kafka**
   - **Cause**: The producer service may not be properly configured or may not be connecting to Kafka.
   - **Solution**:
     - Check the producer service logs to see if there are any connection issues or errors:
       ```bash
       docker logs pipeline-producer
       ```
     - Ensure that the `KAFKA_BROKER_URL` and `INPUT_TOPIC` are correctly set in the `.env` file for the producer.
     - Verify the Kafka broker is up and running by consuming from the topic directly:
       ```bash
       kafka-console-consumer --bootstrap-server localhost:29092 --topic user-login --from-beginning
       ```

### 5. **Service Not Starting or Exiting Unexpectedly**
   - **Cause**: There may be issues with the Docker containers or the environment variables.
   - **Solution**:
     - Check the Docker container logs to identify any errors during startup:
       ```bash
       docker logs <container_name>
       ```
     - Ensure that the `.env` file is properly configured and contains all the required environment variables.
     - Run `docker-compose down` followed by `docker-compose up --build` to rebuild the containers and clear any stale states.

### 6. **Topic Creation Fails**
   - **Cause**: Kafka may fail to create topics automatically if the configuration is incorrect or if permissions are not set correctly.
   - **Solution**:
     - Ensure that the `KAFKA_CREATE_TOPICS` environment variable in the `.env` file lists the correct topics and partition configurations.
     - Manually create the topics using Kafka's CLI:
       ```bash
       kafka-topics.sh --create --topic user-login --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       kafka-topics.sh --create --topic processed-user-login --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       kafka-topics.sh --create --topic user-login-dlq --partitions 1 --replication-factor 1 --bootstrap-server localhost:29092
       ```

### 7. **Kafka Logs Not Showing Consumer Activity**
   - **Cause**: The consumer might be configured to use a manual commit strategy, and the logs may not reflect offset commits.
   - **Solution**:
     - Ensure that the `ENABLE_AUTO_COMMIT` variable is set to `false` for manual offset control, and manually commit offsets in the code when processing is complete.
     - Check that the consumer group is correctly set in the `.env` file with the `CONSUMER_GROUP` variable.

### 8. **Graceful Shutdown Not Working**
   - **Cause**: The consumer may not be properly handling termination signals.
   - **Solution**:
     - Ensure that the shutdown logic is implemented correctly in the Go consumer to handle SIGINT and SIGTERM signals. Example code for graceful shutdown in Go:
       ```go
       sigs := make(chan os.Signal, 1)
       signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
       <-sigs
       // Cleanup and shutdown logic here
       ```

### 9. **Docker Compose Failing to Start Containers**
   - **Cause**: There may be conflicts with port bindings or missing dependencies.
   - **Solution**:
     - Ensure no other services are using the same ports as defined in your `docker-compose.yml` (e.g., `29092` for Kafka).
     - Use `docker-compose logs` to diagnose which service failed to start and why.

For additional support, please refer to the official Kafka documentation or open an issue on the GitHub repository.

## Conclusion

This solution provides a scalable, fault-tolerant real-time data pipeline using Kafka, Docker, and Go. The design ensures efficient message processing with a consumer that can handle retries and handle errors through the Dead Letter Queue. This setup can be easily deployed in production environments with Kubernetes and monitored using tools like Prometheus and Grafana.

For any questions or support, feel free to open an issue on the repository.
