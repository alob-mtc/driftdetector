resource "aws_instance" "example" {
  ami = "ami-0c55b159cbfafe1f0"
  # Missing instance_type but include a dummy attribute to make the block valid
  dummy = "value"
} 