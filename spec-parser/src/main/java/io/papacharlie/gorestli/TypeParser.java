package io.papacharlie.gorestli;

import com.google.common.base.Preconditions;
import com.linkedin.data.schema.ArrayDataSchema;
import com.linkedin.data.schema.DataSchema;
import com.linkedin.data.schema.DataSchemaLocation;
import com.linkedin.data.schema.DataSchemaResolver;
import com.linkedin.data.schema.EnumDataSchema;
import com.linkedin.data.schema.FixedDataSchema;
import com.linkedin.data.schema.MapDataSchema;
import com.linkedin.data.schema.NamedDataSchema;
import com.linkedin.data.schema.RecordDataSchema;
import com.linkedin.data.schema.TyperefDataSchema;
import com.linkedin.data.schema.UnionDataSchema;
import com.linkedin.restli.restspec.CollectionSchema;
import com.linkedin.restli.restspec.ResourceSchema;
import com.linkedin.restli.restspec.RestSpecCodec;
import com.linkedin.restli.tools.snapshot.gen.SnapshotGenerator;
import io.papacharlie.gorestli.json.ComplexKey;
import io.papacharlie.gorestli.json.Enum;
import io.papacharlie.gorestli.json.Fixed;
import io.papacharlie.gorestli.json.GoRestliSpec.DataType;
import io.papacharlie.gorestli.json.Method.PathKey;
import io.papacharlie.gorestli.json.Record;
import io.papacharlie.gorestli.json.Record.Field;
import io.papacharlie.gorestli.json.RestliType;
import io.papacharlie.gorestli.json.StandaloneUnion;
import io.papacharlie.gorestli.json.Typeref;
import java.io.File;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;
import javax.annotation.Nullable;

import static com.linkedin.data.schema.DataSchema.Type.*;
import static io.papacharlie.gorestli.json.RestliType.*;


public class TypeParser {
  private final Map<Identifier, DataType> _dataTypes = new HashMap<>();

  private final DataSchemaResolver _dataSchemaResolver;

  public TypeParser(DataSchemaResolver dataSchemaResolver) {
    _dataSchemaResolver = dataSchemaResolver;
  }

  public void extractDataTypes(ResourceSchema resourceSchema) {
    SnapshotGenerator generator = new SnapshotGenerator(resourceSchema, _dataSchemaResolver);
    for (NamedDataSchema schema : generator.generateModelList()) {
      DataSchemaLocation location = _dataSchemaResolver.nameToDataSchemaLocations().get(schema.getFullName());
      Preconditions.checkNotNull(location, "Could not resolve original location for %s", schema.getFullName());
      File sourceFile = location.getSourceFile();
      switch (schema.getType()) {
        case RECORD:
          parseDataType((RecordDataSchema) schema, sourceFile);
          break;
        case TYPEREF:
          parseDataType((TyperefDataSchema) schema, sourceFile);
          break;
        case ENUM:
          parseDataType((EnumDataSchema) schema, sourceFile);
          break;
        case FIXED:
          parseDataType((FixedDataSchema) schema, sourceFile);
          break;
        default:
          System.err.printf("Don't know what to do with %s%n", schema);
      }
    }
  }

  public Set<DataType> getDataTypes() {
    return Collections.unmodifiableSet(new HashSet<>(_dataTypes.values()));
  }

  public RestliType parseFromRestSpec(String schema) {
    return fromDataSchema(RestSpecCodec.textToSchema(schema, _dataSchemaResolver), null, null, null);
  }

  public PathKey collectionPathKey(String resourceName, String resourceNamespace, CollectionSchema collection,
      File specFile) {
    RestliType pkType;
    if (collection.getIdentifier().hasParams()) {
      RestliType keyType = parseFromRestSpec(collection.getIdentifier().getType());
      Preconditions.checkNotNull(keyType._reference, "Complex key \"key\" must be a record type");
      RestliType paramsType = parseFromRestSpec(collection.getIdentifier().getParams());
      Preconditions.checkNotNull(paramsType._reference, "Complex key \"params\" must be a record type");
      ComplexKey key =
          new ComplexKey(resourceName, resourceNamespace, specFile, keyType._reference, paramsType._reference);
      registerDataType(new DataType(key));
      pkType = key.restliType();
    } else {
      pkType = parseFromRestSpec(collection.getIdentifier().getType());
    }

    return new PathKey(collection.getIdentifier().getName(), pkType);
  }

  private void parseDataType(RecordDataSchema schema, File sourceFile) {
    List<Field> fields = new ArrayList<>();
    for (RecordDataSchema.Field field : schema.getFields()) {
      RestliType fieldRestliType = fromDataSchema(
          field.getType(),
          schema.getNamespace(),
          sourceFile,
          Arrays.asList(schema.getName(), field.getName()));
      boolean optional = field.getOptional();

      fields.add(new Field(
          field.getName(),
          field.getDoc(),
          fieldRestliType,
          optional,
          field.getDefault()));
    }

    registerDataType(new DataType(new Record(schema, sourceFile, fields)));
  }

  private void parseDataType(TyperefDataSchema schema, File sourceFile) {
    DataSchema refSchema = schema.getDereferencedDataSchema();
    RestliType referencedType =
        fromDataSchema(refSchema, schema.getNamespace(), sourceFile, Collections.singletonList(schema.getName()));

    Typeref ref;

    if (referencedType._primitive != null) {
      ref = new Typeref(schema, sourceFile, referencedType._primitive);
    } else if (referencedType._array != null || referencedType._map != null) {
      ref = new Typeref(schema, sourceFile, referencedType._array, referencedType._map);
    } else {
      DataType underlyingType = _dataTypes.get(referencedType._reference);
      if (underlyingType != null && underlyingType.getNamedType() instanceof StandaloneUnion) {
        StandaloneUnion union = (StandaloneUnion) underlyingType.getNamedType();
        _dataTypes.remove(referencedType._reference);
        ref = new Typeref(schema, sourceFile, union._union);
      } else {
        ref = new Typeref(schema, sourceFile, referencedType._reference);
      }
    }

    registerDataType(new DataType(ref));
  }

  private void parseDataType(EnumDataSchema schema, File sourceFile) {
    registerDataType(new DataType(new Enum(schema, sourceFile, schema.getSymbols(), schema.getSymbolDocs())));
  }

  private void parseDataType(FixedDataSchema schema, File sourceFile) {
    registerDataType(new DataType(new Fixed(schema, sourceFile, schema.getSize())));
  }

  private RestliType fromDataSchema(DataSchema schema, @Nullable String namespace, @Nullable File sourceFile,
      @Nullable List<String> hierarchy) {
    if (JAVA_TO_GO_PRIMTIIVE_TYPE.containsKey(schema.getType())) {
      return new RestliType(JAVA_TO_GO_PRIMTIIVE_TYPE.get(schema.getType()));
    }

    switch (schema.getType()) {
      case TYPEREF:
      case RECORD:
      case FIXED:
      case ENUM:
        NamedDataSchema namedSchema = (NamedDataSchema) schema;
        return new RestliType(new Identifier(namedSchema.getNamespace(), namedSchema.getName()));
      case ARRAY:
        RestliType arrayType = fromDataSchema(
            ((ArrayDataSchema) schema).getItems(),
            namespace,
            sourceFile,
            Utils.append(hierarchy, "Array"));
        return new RestliType(arrayType, null);
      case MAP:
        RestliType mapType = fromDataSchema(
            ((MapDataSchema) schema).getValues(),
            namespace,
            sourceFile,
            Utils.append(hierarchy, "Map"));
        return new RestliType(null, mapType);
      case UNION:
        Preconditions.checkState(namespace != null && hierarchy != null,
            "Raw unions not supported outside of records or typerefs");
        UnionDataSchema unionSchema = (UnionDataSchema) schema;

        List<UnionMember> members = unionSchema.getMembers().stream()
            .map(m -> m.getType().getType() == NULL
                ? null
                : new UnionMember(fromDataSchema(m.getType(), namespace, sourceFile, hierarchy), m.getUnionMemberKey()))
            .collect(Collectors.toList());

        String name = hierarchy.stream()
            .map(Utils::exportedIdentifier)
            .collect(Collectors.joining("_"));
        StandaloneUnion union = new StandaloneUnion(name, namespace, sourceFile, members);
        registerDataType(new DataType(union));

        return new RestliType(union.getIdentifier());
      default:
        throw new UnknownTypeException(schema.getType());
    }
  }

  public static class UnknownTypeException extends RuntimeException {
    private UnknownTypeException(DataSchema.Type type) {
      super("Unknown type: " + type);
    }
  }

  private void registerDataType(DataType type) {
    _dataTypes.put(type.getNamedType().getIdentifier(), type);
  }
}
