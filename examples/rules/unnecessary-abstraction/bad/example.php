<?php

function persist($value) {
    return $value * 2;
}
interface Store {
    public function save($value);
}
class FileStore implements Store {
    public function save($value) {
        return persist($value);
    }
}
