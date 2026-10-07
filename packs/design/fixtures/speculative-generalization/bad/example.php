<?php

class FormatOptions {
    public $uppercase = false;
    public $padding = 0;
    public $verbose = false;
}

function format_name($name, FormatOptions $options = null) {
    return trim($name);
}
