  resource "aws_instance" "example" {
    ami                    = "ami-0274f4b62b6ae3bd4"
    instance_type          = "t2.small"
    subnet_id              = "subnet-12345"
    vpc_security_group_ids = ["sg-12345", "sg-0ede820f9c1b2b4bf"]

    tags = {
      Name = "TestInstance"
      Env  = "Test"
    }
  }