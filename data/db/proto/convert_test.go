package proto

import (
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/windmilleng/mish/data"
)

func TestPathConversionUtf8(t *testing.T) {
	op := data.DirOp{Names: []string{"foo", "bar"}}
	recipe := data.Recipe{Op: &op}
	recipeProto, err := RecipeD2P(recipe)
	if err != nil {
		t.Fatal(err)
	}

	actual := recipeProto.Op.(*Recipe_OpDir).OpDir.Name
	if len(actual) != 2 || actual[0].Val.(*Path_Utf8).Utf8 != "foo" {
		t.Fatalf("Bad Domain -> proto conversion. %v", actual)
	}

	recipeString, err := (&jsonpb.Marshaler{}).MarshalToString(recipeProto)
	if err != nil {
		t.Fatal(err)
	}

	if recipeString != `{"inputSnapId":[],"opDir":{"name":["foo","bar"]}}` {
		t.Fatalf("Unexpected recipe json: %s", recipeString)
	}

	recipeProto = &Recipe{}
	err = jsonpb.UnmarshalString(recipeString, recipeProto)
	if err != nil {
		t.Fatal(err)
	}

	recipe, err = RecipeP2D(recipeProto)
	if err != nil {
		t.Fatal(err)
	}

	if recipe.Op.(*data.DirOp).Names[0] != "foo" {
		t.Fatalf("Unexpected recipe: %v", recipe)
	}
}

func TestPathConversionRaw(t *testing.T) {
	op := data.RemoveFileOp{Path: string([]byte{0xAD})}
	recipe := data.Recipe{Op: &op}
	recipeProto, err := RecipeD2P(recipe)
	if err != nil {
		t.Fatal(err)
	}

	actual := recipeProto.Op.(*Recipe_OpRemoveFile).OpRemoveFile.Path
	if len(actual.Val.(*Path_Raw).Raw) != 1 || actual.Val.(*Path_Raw).Raw[0] != 0xAD {
		t.Fatalf("Bad Domain -> proto conversion. %v", actual)
	}

	recipeString, err := (&jsonpb.Marshaler{}).MarshalToString(recipeProto)
	if err != nil {
		t.Fatal(err)
	}

	if recipeString != `{"inputSnapId":[],"opRemoveFile":{"path":{"raw":"rQ=="}}}` {
		t.Fatalf("Unexpected recipe json: %s", recipeString)
	}

	recipeProto = &Recipe{}
	err = jsonpb.UnmarshalString(recipeString, recipeProto)
	if err != nil {
		t.Fatal(err)
	}

	recipe, err = RecipeP2D(recipeProto)
	if err != nil {
		t.Fatal(err)
	}

	if recipe.Op.(*data.RemoveFileOp).Path != "\xAD" {
		t.Fatalf("Unexpected recipe: %v", recipe)
	}
}
