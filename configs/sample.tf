resource "aws_instance" "example" {
  ami                         = "ami-00a929b66ed6e0de6"
  instance_type               = "t2.micro"
  subnet_id                   = "subnet-0b98e7dc732b59e7a"
  vpc_security_group_ids      = ["sg-0ede820f9c1b2b4bf"]

  tags = {
    Name = "TestInstance"
    Env  = "Staging"
  }
}
