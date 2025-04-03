resource "aws_instance" "example" {
  ami                    = "ami-0c55b159cbfafe1f0"
  instance_type          = "t2.micro"
  subnet_id              = "subnet-12345"
  vpc_security_group_ids = ["sg-12345", "sg-67890"]

  tags = {
    Name = "TestInstance"
    Env  = "Test"
  }
} 